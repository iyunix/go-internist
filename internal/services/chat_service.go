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

	// Save user message and bump chat updated_at so it surfaces in history.
	userMessage := &domain.Message{ChatID: chatID, Role: "user", Content: prompt}
	if _, err := s.messageRepo.Create(ctx, userMessage); err != nil {
		return fmt.Errorf("failed to store user message: %w", err)
	}
	_ = s.chatRepo.TouchUpdatedAt(ctx, chatID)

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

	// Persist assistant reply asynchronously and bump chat updated_at.
	go func() {
		if fullReply.Len() > 0 {
			assistantMessage := &domain.Message{ChatID: chatID, Role: "assistant", Content: fullReply.String()}
			if _, err := s.messageRepo.Create(context.Background(), assistantMessage); err != nil {
				log.Printf("Failed to save assistant message: %v", err)
			}
			_ = s.chatRepo.TouchUpdatedAt(context.Background(), chatID)
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

// buildFinalPrompt creates a Markdown-only instruction so the model returns clean Markdown.
func (s *ChatService) buildFinalPrompt(contextJSON, question string) string {
    if strings.TrimSpace(contextJSON) == "" {
        contextJSON = "[]"
    }

    return fmt.Sprintf(`SYSTEM:
You are "Internist", an expert medical assistant. Return the answer in Markdown ONLY.
- Output must be valid Markdown with headings (#, ##, ###), paragraphs, bullet/numbered lists, and tables where helpful.
- Do NOT return JSON, code fences with json, or any wrapper text before or after the Markdown.
- Do NOT include any system or policy text in the output.
- If content is insufficient, write a brief Markdown section explaining the limitation.
- Keep clinical guidance precise, concise, and structured for fast scanning.

STYLE:
- Start with a clear H1 or H2 title for the topic.
- Use short paragraphs, bullet points, and subheadings to organize content.
- Use tables for concise comparisons (e.g., dosing, side effects, labs).
- If citing context, add a final "## References" section listing relevant sources from CONTEXT (by file name or brief identifiers). Do not invent sources.
- Do not include personal data or PHI.

CONTEXT (JSON array of chunks to use as your sole evidence base):
%s

QUESTION:
%s
`, contextJSON, question)
}


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



// ExtractSourceTitles extracts unique document titles from Pinecone matches
func (s *ChatService) ExtractSourceTitles(matches []*pinecone.ScoredVector) []string {
    var sources []string
    seen := make(map[string]bool)

    for _, match := range matches {
        if match == nil || match.Vector == nil || match.Vector.Metadata == nil {
            continue
        }
        
        md := match.Vector.Metadata.GetFields()
        
        // Try to get title from different metadata fields
        var title string
        
        // Priority: source_file > section_heading > chunk ID
        if f, ok := md["source_file"]; ok {
            title = strings.TrimSpace(f.GetStringValue())
            // Clean up the filename - remove extension and path
            title = strings.TrimSuffix(title, ".md")
            title = strings.TrimSuffix(title, "_Drug_information")
            title = strings.ReplaceAll(title, "_", " ")
        }
        
        if title == "" {
            if f, ok := md["section_heading"]; ok {
                title = strings.TrimSpace(f.GetStringValue())
            }
        }
        
        if title == "" {
            title = match.Vector.Id
        }
        
        // Add unique titles only
        if title != "" && !seen[title] {
            sources = append(sources, title)
            seen[title] = true
        }
    }
    
    return sources
}

// StreamChatMessageWithSources - Enhanced version that sends sources via callback
func (s *ChatService) StreamChatMessageWithSources(
    ctx context.Context,
    userID, chatID uint,
    prompt string,
    onDelta func(token string) error,
    onSources func(sources []string),
) error {
    chat, err := s.chatRepo.FindByID(ctx, chatID)
    if err != nil || chat.UserID != userID {
        return errors.New("chat not found or unauthorized")
    }

    // Save user message
    userMessage := &domain.Message{ChatID: chatID, Role: "user", Content: prompt}
    if _, err := s.messageRepo.Create(ctx, userMessage); err != nil {
        return fmt.Errorf("failed to store user message: %w", err)
    }
    _ = s.chatRepo.TouchUpdatedAt(ctx, chatID)

    // Get embedding and query Pinecone
    embedding, err := s.aiService.CreateEmbedding(ctx, prompt)
    if err != nil {
        return fmt.Errorf("failed to create embedding: %w", err)
    }
    matches, err := s.pineconeService.QuerySimilar(ctx, embedding, s.retrievalTopK)
    if err != nil {
        return fmt.Errorf("failed to query pinecone: %w", err)
    }

    // Extract and send source titles to frontend
    sources := s.ExtractSourceTitles(matches)
    if len(sources) > 0 && onSources != nil {
        onSources(sources)
    }

    // Build context and generate response
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

    // Save assistant message
    go func() {
        if fullReply.Len() > 0 {
            assistantMessage := &domain.Message{ChatID: chatID, Role: "assistant", Content: fullReply.String()}
            if _, err := s.messageRepo.Create(context.Background(), assistantMessage); err != nil {
                log.Printf("Failed to save assistant message: %v", err)
            }
            _ = s.chatRepo.TouchUpdatedAt(context.Background(), chatID)
        }
    }()
    return nil
}