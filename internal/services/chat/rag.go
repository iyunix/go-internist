package chat

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

// contextEntry represents a normalized RAG context chunk
type contextEntry struct {
	ChunkID        string `json:"chunk_id"`
	SourceFile     string `json:"source_file"`
	SectionHeading string `json:"section_heading"`
	KeyTakeaways   string `json:"key_takeaways"`
	Text           string `json:"text"`
	Similarity     string `json:"similarity"`
}

// RAGService handles building structured context from Pinecone embeddings
type RAGService struct {
	config *Config
	logger Logger
}

// NewRAGService initializes the RAG service
func NewRAGService(config *Config, logger Logger) *RAGService {
	return &RAGService{
		config: config,
		logger: logger,
	}
}

// BuildContext converts Pinecone matches into a JSON array of context entries
// Returns both the JSON string and the structured entries for later use in references
func (r *RAGService) BuildContext(matches []*pinecone.ScoredVector) (string, []contextEntry) {
	r.logger.Info("building RAG context", "matches_count", len(matches))

	// Sort matches by descending similarity score
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

	entries := make([]contextEntry, 0, len(matches))
	for i, match := range matches {
		if match == nil || match.Vector == nil || match.Vector.Metadata == nil {
			continue
		}
		entry := r.extractContextEntry(match, i)
		if entry.ChunkID != "" {
			entries = append(entries, entry)
		}
	}

	contextJSON := r.serializeContext(entries)
	r.logger.Info("RAG context built", "entries_count", len(entries))
	return contextJSON, entries
}

// extractContextEntry parses a single Pinecone match into a structured context entry
func (r *RAGService) extractContextEntry(match *pinecone.ScoredVector, index int) contextEntry {
	md := match.Vector.Metadata.GetFields()

	entry := contextEntry{
		ChunkID:    match.Vector.Id,
		Similarity: strconv.FormatFloat(float64(match.Score), 'f', 6, 64),
	}

	if entry.ChunkID == "" {
		entry.ChunkID = fmt.Sprintf("C%03d", index+1)
	}

	if f, ok := md["source_file"]; ok {
		entry.SourceFile = f.GetStringValue()
	}
	if f, ok := md["section_heading"]; ok {
		entry.SectionHeading = f.GetStringValue()
	}
	if f, ok := md["key_takeaways"]; ok {
		entry.KeyTakeaways = f.GetStringValue()
	}
	if f, ok := md["text"]; ok {
		entry.Text = f.GetStringValue()
	}

	log.Printf("[RAG Context] Chunk %d Source: %s Id: %s", index+1, entry.SourceFile, entry.ChunkID)
	return entry
}

// serializeContext converts structured entries to a JSON-safe string
func (r *RAGService) serializeContext(entries []contextEntry) string {
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
			esc(e.ChunkID), esc(e.SourceFile), esc(e.SectionHeading),
			esc(e.KeyTakeaways), esc(e.Text), e.Similarity)
	}

	b.WriteString("\n]")
	return b.String()
}

// BuildPrompt generates a medical AI prompt from context JSON, user question, and entries
// The entries are used to deterministically generate a References section
func (r *RAGService) BuildPrompt(contextJSON, question string, entries []contextEntry) string {
	if strings.TrimSpace(contextJSON) == "" {
		contextJSON = "[]"
	}

	// Build deterministic References section from retrieved chunks
	var references strings.Builder
	if len(entries) > 0 {
		references.WriteString("\n## References\n")
		for _, e := range entries {
			if e.SourceFile != "" {
				references.WriteString(fmt.Sprintf("- %s\n", e.SourceFile))
			}
		}
	}

	return fmt.Sprintf(`You Are a medical assitant
	# Context
	%s
	# Question
	%s
	# Instructions
	- Use only the above context to answer the question.
	- Return your answer in valid Markdown (no JSON, no extra explanations).
	- If info is missing, clearly state what can't be answered.
	- Your reply must be in English and concise, organized with headings, bullets, or tables as needed.
	%s
`, contextJSON, question, references.String())
}