// G:\go_internist\internal\services\chat\rag.go
package chat

import (
    "fmt"
    "log"
    "sort"
    "strconv"
    "strings"

    "github.com/qdrant/go-client/qdrant"
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

// RAGService handles building structured context from Qdrant embeddings
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

// BuildContext converts Qdrant matches into a JSON array of context entries
// Returns both the JSON string and the structured entries for later use in references
func (r *RAGService) BuildContext(matches []*qdrant.ScoredPoint) (string, []contextEntry) {
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
        if match == nil || match.Payload == nil {
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

// extractContextEntry parses a single Qdrant match into a structured context entry
func (r *RAGService) extractContextEntry(match *qdrant.ScoredPoint, index int) contextEntry {
    // Extract point ID
    pointID := r.extractPointID(match)
    
    entry := contextEntry{
        ChunkID:    pointID,
        Similarity: strconv.FormatFloat(float64(match.Score), 'f', 6, 64),
    }

    if entry.ChunkID == "" {
        entry.ChunkID = fmt.Sprintf("C%03d", index+1)
    }

    // Extract metadata from Qdrant payload (now using *qdrant.Value type)
    if match.Payload != nil {
        if sourceFile := r.getStringFromQdrantPayload(match.Payload, "source_file"); sourceFile != "" {
            entry.SourceFile = sourceFile
        }
        if sectionHeading := r.getStringFromQdrantPayload(match.Payload, "section_heading"); sectionHeading != "" {
            entry.SectionHeading = sectionHeading
        }
        if text := r.getStringFromQdrantPayload(match.Payload, "text"); text != "" {
            entry.Text = text
        }
    }

    log.Printf("[RAG Context] Chunk %d Source: %s Id: %s", index+1, entry.SourceFile, entry.ChunkID)
    return entry
}

// extractPointID safely extracts the point ID from a Qdrant ScoredPoint
func (r *RAGService) extractPointID(point *qdrant.ScoredPoint) string {
    if point.Id == nil {
        return ""
    }
    
    switch id := point.Id.PointIdOptions.(type) {
    case *qdrant.PointId_Num:
        return strconv.FormatUint(id.Num, 10)
    case *qdrant.PointId_Uuid:
        return id.Uuid
    default:
        return ""
    }
}

// getStringFromQdrantPayload safely extracts string values from Qdrant payload (using *qdrant.Value)
func (r *RAGService) getStringFromQdrantPayload(payload map[string]*qdrant.Value, key string) string {
    if payload == nil {
        return ""
    }
    
    value, ok := payload[key]
    if !ok || value == nil {
        return ""
    }
    
    switch v := value.Kind.(type) {
    case *qdrant.Value_StringValue:
        return v.StringValue
    case *qdrant.Value_IntegerValue:
        return strconv.FormatInt(v.IntegerValue, 10)
    case *qdrant.Value_DoubleValue:
        return strconv.FormatFloat(v.DoubleValue, 'f', -1, 64)
    case *qdrant.Value_BoolValue:
        return strconv.FormatBool(v.BoolValue)
    default:
        return ""
    }
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

        fmt.Fprintf(&b, `  {"chunk_id":"%s","source_file":"%s","section_heading":"%s",,"text":"%s","similarity":%s}`,
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
