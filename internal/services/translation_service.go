// G:\go_internist\internal\services\translation_service.go
package services

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "regexp"
    "strings"
    "time"
    "github.com/iyunix/go-internist/internal/domain"  // ← ADD THIS LINE

)

type TranslationService struct {
    apiKey  string
    baseURL string
    model   string
    client  *http.Client
    logger  Logger
}

func NewTranslationService(apiKey, model string, logger Logger) *TranslationService {
    if model == "" {
        model = "gemini-2.5-flash-lite"
    }
    return &TranslationService{
        apiKey:  apiKey,
        baseURL: "https://api.avalai.ir/v1",
        model:   model,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        logger: logger,
    }
}

func (ts *TranslationService) IsPurelyEnglish(text string) bool {
    punctuationRegex := regexp.MustCompile(`[.,!?'"()\-\s\d]+`)
    cleanText := punctuationRegex.ReplaceAllString(text, "")
    if len(cleanText) == 0 {
        return true
    }
    for _, r := range cleanText {
        if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
            return false
        }
    }
    return true
}

func (ts *TranslationService) ContainsPersian(text string) bool {
    persianRegex := regexp.MustCompile(`\p{Arabic}`)
    return persianRegex.MatchString(text)
}

func (ts *TranslationService) NeedsTranslation(text string) bool {
    text = strings.TrimSpace(text)
    if len(text) == 0 {
        return false
    }
    if ts.IsPurelyEnglish(text) {
        ts.logger.Debug("Text is purely English, no translation needed", "text", text)
        return false
    }
    if ts.ContainsPersian(text) {
        ts.logger.Debug("Text contains Persian characters, translation needed", "text", text)
        return true
    }
    ts.logger.Debug("Text contains non-English characters, translation needed", "text", text)
    return true
}

func (ts *TranslationService) TranslateToEnglish(ctx context.Context, text string) (string, error) {
    ts.logger.Info("Starting translation", "text", text)

    systemPrompt := "Translate the following Persian medical text to clear and precise English. Provide only the direct translation without any explanations, comments, or additional information."
    userPrompt := text

    requestBody := map[string]interface{}{
        "model": ts.model,
        "messages": []map[string]string{
            {"role": "system", "content": systemPrompt},
            {"role": "user", "content": userPrompt},
        },
    }

    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        ts.logger.Error("Failed to marshal translation request", "error", err)
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", ts.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
    if err != nil {
        ts.logger.Error("Failed to create translation request", "error", err)
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+ts.apiKey)

    resp, err := ts.client.Do(req)
    if err != nil {
        ts.logger.Warn("Translation API call failed", "error", err)
        return "", err
    }
    defer resp.Body.Close()

    bodyBytes, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        ts.logger.Error("Translation API returned error", "status_code", resp.StatusCode, "body", string(bodyBytes))
        return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
    }

    // Parse OpenAI-compatible response
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }

    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        ts.logger.Error("Failed to decode translation response", "error", err)
        return "", fmt.Errorf("failed to decode response: %w", err)
    }

    if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
        ts.logger.Warn("Translation API returned empty result", "original", text)
        return "", fmt.Errorf("translation returned empty result")
    }

    translation := strings.TrimSpace(result.Choices[0].Message.Content)
    ts.logger.Info("Translation completed successfully", "original", text, "translated", translation)

    return translation, nil
}

// Add this NEW method to your existing translation_service.go (keep all existing methods unchanged)

// UPDATED NEW METHOD - Process ALL queries (Persian + English) with context awareness
func (ts *TranslationService) TranslateWithMedicalContext(
    ctx context.Context,
    currentQuery string,
    conversationHistory []domain.Message,
) (embeddingQuery string, llmQuery string, err error) {
    ts.logger.Info("Starting context-aware processing", 
        "current_query", currentQuery,
        "history_length", len(conversationHistory))

    // Step 1: Translate if needed (Persian → English)
    translatedCurrent := currentQuery
    if ts.NeedsTranslation(currentQuery) {
        var err error
        translatedCurrent, err = ts.TranslateToEnglish(ctx, currentQuery)
        if err != nil {
            ts.logger.Error("Failed to translate current query", "error", err)
            return "", "", err
        }
    }

    // Step 2: Prepare conversation context (last few exchanges - max 3 pairs)
    contextMessages := ts.prepareConversationContext(conversationHistory, 3)
    
    // Step 3: ALWAYS generate focused embedding query (for Persian AND English queries)
    if len(contextMessages) > 0 {
        embeddingQuery, err = ts.generateFocusedEmbeddingQuery(ctx, translatedCurrent, contextMessages)
        if err != nil {
            ts.logger.Warn("Failed to generate focused embedding query, using translated query", "error", err)
            embeddingQuery = translatedCurrent
        }
    } else {
        // No context, use translated query as-is
        embeddingQuery = translatedCurrent
    }

    // Step 4: LLM query is the translated current query (context added separately)
    llmQuery = translatedCurrent

    ts.logger.Info("Context-aware processing completed",
        "original", currentQuery,
        "translated", translatedCurrent,
        "embedding_query", embeddingQuery,
        "embedding_length", len(embeddingQuery))

    return embeddingQuery, llmQuery, nil
}

// prepareConversationContext formats recent conversation for context analysis (max 3 pairs)
func (ts *TranslationService) prepareConversationContext(messages []domain.Message, maxPairs int) string {
    if len(messages) == 0 || maxPairs <= 0 {
        return ""
    }

    var contextParts []string
    count := 0
    
    // Process messages to get recent context (already limited to 3 pairs by caller)
    for _, msg := range messages {
        if msg.MessageType == "internal_context" || msg.MessageType == "assistant" {
            // Limit each message to avoid context bloat
            content := msg.Content
            if len(content) > 150 {
                content = content[:150] + "..."
            }
            
            role := "User"
            if msg.MessageType == "assistant" {
                role = "Assistant"
            }
            
            contextParts = append(contextParts, fmt.Sprintf("%s: %s", role, content))
            count++
            if count >= maxPairs*2 { // Max 6 messages (3 pairs)
                break
            }
        }
    }
    
    return strings.Join(contextParts, "\n")
}

// generateFocusedEmbeddingQuery uses AI to create topic-focused queries for better RAG retrieval
func (ts *TranslationService) generateFocusedEmbeddingQuery(ctx context.Context, currentQuery, conversationContext string) (string, error) {
    systemPrompt := `You are a medical query optimizer for RAG systems. Create focused, specific medical queries for optimal chunk retrieval.

CRITICAL RULES:
1. ALWAYS focus on the CURRENT query's intent, not previous topics
2. If current query asks about TREATMENT/DRUGS → focus on treatment
3. If current query asks about COMPLICATIONS → focus on complications  
4. If current query asks about DIAGNOSIS → focus on diagnosis
5. Include the disease name + current query intent
6. Keep queries concise but medically specific
7. Always output in English

Examples:
Current: "treatment drugs for scleroderma", Context: "complications" 
→ Output: "scleroderma treatment medications drugs"

Current: "complications of GPA", Context: "treatment discussion"
→ Output: "granulomatosis with polyangiitis complications"

Current: "side effects", Context: "methotrexate arthritis"
→ Output: "methotrexate side effects"`

    userPrompt := fmt.Sprintf(`Current query: "%s"
Recent context: %s

Focus on the CURRENT query intent. Create focused medical query:`, currentQuery, conversationContext)

    // Rest of the method remains the same...
    requestBody := map[string]interface{}{
        "model": ts.model,
        "messages": []map[string]string{
            {"role": "system", "content": systemPrompt},
            {"role": "user", "content": userPrompt},
        },
    }

    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", ts.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+ts.apiKey)

    resp, err := ts.client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    bodyBytes, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
    }

    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }

    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return "", fmt.Errorf("failed to decode response: %w", err)
    }

    if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
        return "", fmt.Errorf("empty AI response for query optimization")
    }

    focusedQuery := strings.TrimSpace(result.Choices[0].Message.Content)
    
    // Clean up common AI response patterns
    focusedQuery = strings.TrimPrefix(focusedQuery, "Output: ")
    focusedQuery = strings.TrimPrefix(focusedQuery, "Focused query: ")
    focusedQuery = strings.Trim(focusedQuery, `"`)

    return focusedQuery, nil
}
