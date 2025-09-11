//G:\go_internist\internal\services\chat\context.go
package chat

import (
    "fmt"
    "strings"
    "unicode/utf8"
)

// ContextHelper provides utilities for managing chat context and text processing
type ContextHelper struct {
    config *Config
    logger Logger
}

// NewContextHelper creates a new context helper with configuration
func NewContextHelper(config *Config, logger Logger) *ContextHelper {
    return &ContextHelper{
        config: config,
        logger: logger,
    }
}

// TruncateText safely truncates a UTF-8 string to maxLen runes, preserving character integrity
func (ch *ContextHelper) TruncateText(input string, maxLen int) string {
    if input == "" || maxLen <= 0 {
        return ""
    }

    if utf8.RuneCountInString(input) <= maxLen {
        return input
    }

    var b strings.Builder
    count := 0

    for _, r := range input {
        if count >= maxLen {
            break
        }
        b.WriteRune(r)
        count++
    }

    return b.String()
}

// EscapeJSON escapes a string for safe JSON serialization
func (ch *ContextHelper) EscapeJSON(input string) string {
    escaped := strings.ReplaceAll(input, `\`, `\\`)
    escaped = strings.ReplaceAll(escaped, `"`, `\"`)
    escaped = strings.ReplaceAll(escaped, "\n", `\n`)
    escaped = strings.ReplaceAll(escaped, "\r", `\r`)
    escaped = strings.ReplaceAll(escaped, "\t", `\t`)
    return escaped
}

// CleanWhitespace normalizes whitespace in text for better processing
func (ch *ContextHelper) CleanWhitespace(input string) string {
    // Replace multiple consecutive whitespaces with a single space
    lines := strings.Fields(input)
    return strings.Join(lines, " ")
}

// ValidateContextSize checks if context fits within token limits
func (ch *ContextHelper) ValidateContextSize(contextJSON string) bool {
    // Rough estimation: 1 token ≈ 4 characters for English text
    estimatedTokens := len(contextJSON) / 4
    return estimatedTokens <= ch.config.ContextMaxTokens
}

// TruncateContext intelligently truncates context to fit token limits
func (ch *ContextHelper) TruncateContext(contextJSON string) string {
    if ch.ValidateContextSize(contextJSON) {
        return contextJSON
    }

    ch.logger.Info(
        "truncating context for token limits",
        "original_length", len(contextJSON),
        "max_tokens", ch.config.ContextMaxTokens,
    )

    // Calculate target length (assuming 4 chars per token)
    targetLength := ch.config.ContextMaxTokens * 4

    // Try to truncate at a reasonable boundary (end of JSON objects)
    if len(contextJSON) > targetLength {
        truncated := contextJSON[:targetLength]

        // Find last complete JSON object boundary
        lastBrace := strings.LastIndex(truncated, "}")
        if lastBrace > 0 {
            truncated = truncated[:lastBrace+1]
        }

        // Ensure valid JSON array closing
        if !strings.HasSuffix(truncated, "]") {
            truncated += "\n]"
        }

        return truncated
    }

    return contextJSON
}

// ExtractKeywords extracts important keywords from medical text
func (ch *ContextHelper) ExtractKeywords(text string, maxKeywords int) []string {
    if text == "" || maxKeywords <= 0 {
        return nil
    }

    // Simple keyword extraction - can be enhanced with NLP
    words := strings.Fields(strings.ToLower(text))

    // Medical-specific keyword filtering
    medicalKeywords := make(map[string]int)
    medicalIndicators := []string{
        "symptom", "diagnosis", "treatment", "patient", "condition",
        "disease", "medication", "therapy", "clinical", "medical",
        "syndrome", "disorder", "infection", "chronic", "acute",
    }

    for _, word := range words {
        // Clean word
        word = strings.Trim(word, ".,!?;:()")

        // Skip short words
        if len(word) < 3 {
            continue
        }

        // Prioritize medical terms
        for _, indicator := range medicalIndicators {
            if strings.Contains(word, indicator) {
                medicalKeywords[word] += 3
                break
            }
        }

        medicalKeywords[word]++
    }

    // Convert to slice
    var keywords []string
    for word := range medicalKeywords {
        keywords = append(keywords, word)
    }

    // Limit to maxKeywords
    if len(keywords) > maxKeywords {
        keywords = keywords[:maxKeywords]
    }

    return keywords
}

// FormatPromptSection formats a section of the prompt with proper spacing
func (ch *ContextHelper) FormatPromptSection(title, content string) string {
    if content == "" {
        return ""
    }

    var b strings.Builder
    b.WriteString(fmt.Sprintf("\n%s:\n", strings.ToUpper(title)))
    b.WriteString(content)
    b.WriteString("\n")

    return b.String()
}

// CalculateContextWeight estimates the importance weight of context chunks
func (ch *ContextHelper) CalculateContextWeight(similarity float64, sourceRelevance int) float64 {
    // Combine similarity score with source relevance
    // similarity: 0.0-1.0, sourceRelevance: 1-5
    baseWeight := similarity
    relevanceBonus := float64(sourceRelevance) * 0.1

    return baseWeight + relevanceBonus
}

// SanitizeForPrompt removes potentially problematic characters from prompt text
func (ch *ContextHelper) SanitizeForPrompt(input string) string {
    // Remove or replace characters that might interfere with prompt processing
    sanitized := strings.ReplaceAll(input, "\x00", "") // Remove null bytes
    sanitized = strings.ReplaceAll(sanitized, "\r\n", "\n") // Normalize line endings
    sanitized = strings.ReplaceAll(sanitized, "\r", "\n")   // Mac-style line endings

    // Limit excessive newlines
    for strings.Contains(sanitized, "\n\n\n") {
        sanitized = strings.ReplaceAll(sanitized, "\n\n\n", "\n\n")
    }

    return sanitized
}

// BuildContextMetadata creates metadata summary for context
func (ch *ContextHelper) BuildContextMetadata(numChunks int, avgSimilarity float64, sources []string) string {
    return fmt.Sprintf(
        "Context: %d chunks, avg similarity: %.2f, sources: %d unique",
        numChunks, avgSimilarity, len(sources),
    )
}

// ---------------- Package-level utility functions ----------------

// TruncateText is a package-level helper for simple text truncation
func TruncateText(input string, maxLen int) string {
    if input == "" || maxLen <= 0 {
        return ""
    }
    if utf8.RuneCountInString(input) <= maxLen {
        return input
    }

    var b strings.Builder
    count := 0

    for _, r := range input {
        if count >= maxLen {
            break
        }
        b.WriteRune(r)
        count++
    }

    return b.String()
}

// EscapeJSON is a package-level helper for JSON escaping
func EscapeJSON(input string) string {
    escaped := strings.ReplaceAll(input, `\`, `\\`)
    escaped = strings.ReplaceAll(escaped, `"`, `\"`)
    escaped = strings.ReplaceAll(escaped, "\n", `\n`)
    return escaped
}

// CleanFilename cleans up filenames for display
func CleanFilename(filename string) string {
    cleaned := strings.TrimSuffix(filename, ".md")
    cleaned = strings.TrimSuffix(cleaned, "_Drug_information")
    cleaned = strings.ReplaceAll(cleaned, "_", " ")
    return strings.TrimSpace(cleaned)
}
