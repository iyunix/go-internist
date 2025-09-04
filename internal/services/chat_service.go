// File: internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/repository"
	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type ChatService struct {
	chatRepo        repository.ChatRepository
	messageRepo     repository.MessageRepository
	aiService       *AIService
	pineconeService *PineconeService
	retrievalTopK   int
}

func NewChatService(
	chatRepo repository.ChatRepository,
	messageRepo repository.MessageRepository,
	aiService *AIService,
	pineconeService *PineconeService,
	retrievalTopK int,
) *ChatService {
	if retrievalTopK <= 0 {
		retrievalTopK = 8
	}
	return &ChatService{
		chatRepo:        chatRepo,
		messageRepo:     messageRepo,
		aiService:       aiService,
		pineconeService: pineconeService,
		retrievalTopK:   retrievalTopK,
	}
}

// CreateChat creates a new chat record in the database.
func (s *ChatService) CreateChat(ctx context.Context, userID uint, title string) (*domain.Chat, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("chat title cannot be empty")
	}
	if len(title) > 100 {
		title = title[:100]
	}

	newChat := &domain.Chat{
		UserID: userID,
		Title:  title,
	}

	createdChat, err := s.chatRepo.Create(ctx, newChat)
	if err != nil {
		log.Printf("[ChatService] Failed to create chat for user %d: %v", userID, err)
		return nil, fmt.Errorf("could not create chat: %w", err)
	}
	return createdChat, nil
}

// StreamChatMessage handles the entire RAG and streaming process.
func (s *ChatService) StreamChatMessage(
	ctx context.Context,
	userID, chatID uint,
	prompt string,
	onDelta func(token string) error,
) error {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return errors.New("chat not found or unauthorized")
	}

	userMessage := &domain.Message{ChatID: chatID, Role: "user", Content: prompt}
	if _, err := s.messageRepo.Create(ctx, userMessage); err != nil {
		return fmt.Errorf("failed to store user message: %w", err)
	}

	embedding, err := s.aiService.CreateEmbedding(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}
	matches, err := s.pineconeService.QuerySimilar(ctx, embedding, s.retrievalTopK)
	if err != nil {
		return fmt.Errorf("failed to query pinecone: %w", err)
	}

	// Build a normalized, stable, chunked CONTEXT to enable precise citations.
	contextJSON := s.buildContextJSON(matches)

	finalPrompt := s.buildFinalPrompt(contextJSON, prompt)

	var fullReply strings.Builder
	streamErr := s.aiService.StreamCompletion(ctx, "jabir-400b", finalPrompt, func(token string) error {
		fullReply.WriteString(token)
		return onDelta(token)
	})
	if streamErr != nil {
		log.Printf("[ChatService] Error during AI stream: %v", streamErr)
		return streamErr
	}

	// Persist assistant reply asynchronously.
	go func() {
		if fullReply.Len() > 0 {
			assistantMessage := &domain.Message{ChatID: chatID, Role: "assistant", Content: fullReply.String()}
			if _, err := s.messageRepo.Create(context.Background(), assistantMessage); err != nil {
				log.Printf("Failed to save assistant message: %v", err)
			}
		}
	}()
	return nil
}

// buildContextJSON converts Pinecone matches into a compact JSON array text blob for the prompt.
func (s *ChatService) buildContextJSON(matches []*pinecone.ScoredVector) string {
	type ctxEntry struct {
		ChunkID        string
		SourceFile     string
		SectionHeading string
		KeyTakeaways   string
		Text           string
		Similarity     string
	}

	entries := make([]ctxEntry, 0, len(matches))
	// Sort matches by descending score for consistent ordering.
	sort.Slice(matches, func(i, j int) bool {
		var is, js float32
		if matches[i] != nil {
			is = matches[i].Score
		}
		if matches[j] != nil {
			js = matches[j].Score
		}
		return is > js
	})

	for i, match := range matches {
		if match == nil || match.Vector == nil || match.Vector.Metadata == nil {
			continue
		}
		md := match.Vector.Metadata.GetFields()
		source := ""
		section := ""
		takeaway := ""
		content := ""
		if f, ok := md["source_file"]; ok {
			source = f.GetStringValue()
		}
		if f, ok := md["section_heading"]; ok {
			section = f.GetStringValue()
		}
		if f, ok := md["key_takeaways"]; ok {
			takeaway = f.GetStringValue()
		}
		if f, ok := md["text"]; ok {
			content = f.GetStringValue()
		}

		// Use vector ID as chunk_id (aligns with working Pinecone logic).
		chunkID := match.Vector.Id
		if strings.TrimSpace(chunkID) == "" {
			chunkID = fmt.Sprintf("C%03d", i+1)
		}

		// Convert similarity score to string.
		sim := strconv.FormatFloat(float64(match.Score), 'f', 6, 64)

		log.Printf("[RAG Context] Chunk %d Source: %s Id: %s", i+1, source, chunkID)

		entries = append(entries, ctxEntry{
			ChunkID:        chunkID,
			SourceFile:     source,
			SectionHeading: section,
			KeyTakeaways:   takeaway,
			Text:           content,
			Similarity:     sim,
		})
	}

	// Serialize into a compact JSON-like text for the prompt.
	var b strings.Builder
	b.WriteString("[\n")
	for i, e := range entries {
		if i > 0 {
			b.WriteString(",\n")
		}
		esc := func(s string) string {
			s = strings.ReplaceAll(s, `\`, `\\`)
			s = strings.ReplaceAll(s, `"`, `\"`)
			s = strings.ReplaceAll(s, "\n", `\n`)
			return s
		}
		fmt.Fprintf(&b, `  {"chunk_id":"%s","source_file":"%s","section_heading":"%s","key_takeaways":"%s","text":"%s","similarity":%s}`,
			esc(e.ChunkID), esc(e.SourceFile), esc(e.SectionHeading), esc(e.KeyTakeaways), esc(e.Text), e.Similarity)
	}
	b.WriteString("\n]")
	return b.String()
}

// buildFinalPrompt creates a strict instruction for the model to return JSON in a fixed schema.
func (s *ChatService) buildFinalPrompt(contextJSON, question string) string {
	if strings.TrimSpace(contextJSON) == "" {
		contextJSON = "[]"
	}
	return fmt.Sprintf(`SYSTEM:
You are "Internist", an expert medical assistant. You MUST return STRICT JSON ONLY. 
- Begin with '{' and end with '}' as the very first and last characters. 
- Do NOT include any text, explanations, code fences, or commentary outside the JSON.
- Do NOT break inside JSON keys or values during streaming. 
- Do NOT add extra fields or omit required fields. 
- Do NOT use null. Use empty arrays [] or empty strings "" if needed.
- No trailing commas in any array or object.

YOUR RESPONSE MUST EXACTLY MATCH THIS SCHEMA:
{
  "items": [
    {
      "title": "string",
      "answer_md": "string",
      "additional_md": ["string"],
      "citations": [
        {
          "chunk_id": "string",
          "source_file": "string",
          "section_heading": "string",
          "quote": "string",
          "similarity": "string"
        }
      ],
      "tags": ["string"]
    }
  ],
  "sources": [
    { "source_file": "string", "display_name": "string", "citation_count": 0 }
  ],
  "meta": {
    "no_answer_reason": "string|optional",
    "notes": "string|optional"
  }
}

POLICY:
- Use ONLY the provided CONTEXT. Do NOT invent or use outside knowledge.
- If CONTEXT is insufficient, return: "items": [] and set "meta.no_answer_reason" with a short explanation.
- For every factual claim in answer_md, include at least one citation. When synthesizing from multiple chunks, include multiple citations.
- Copy chunk_id and source_file EXACTLY from CONTEXT. Do NOT renumber or rename them.
- Quotes in citations must be short (5–30 words) and directly from the 'text' or 'key_takeaways' in CONTEXT.
- Tags should be relevant keywords (e.g., drug name, topic).

CONTEXT (JSON array of chunks):
%s

QUESTION:
%s
`, contextJSON, question)
}


// --- THIS FUNCTION SIGNATURE IS NOW CORRECTED ---
func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]domain.Chat, error) {
	return s.chatRepo.FindByUserID(ctx, userID)
}
func (s *ChatService) GetChatMessages(ctx context.Context, userID, chatID uint) ([]domain.Message, error) {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return nil, errors.New("chat not found or unauthorized")
	}
	return s.messageRepo.FindByChatID(ctx, chatID)
}
func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID uint) error {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil || chat.UserID != userID {
		return errors.New("chat not found or unauthorized")
	}
	return s.chatRepo.Delete(ctx, chatID, userID)
}
func (s *ChatService) AddChatMessage(ctx context.Context, userID, chatID uint, content string) (string, domain.Chat, error) {
	return "This is the non-streaming endpoint.", domain.Chat{}, nil
}
