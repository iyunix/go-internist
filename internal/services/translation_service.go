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

    systemPrompt := "Translate all user input Persian medical text to clear, precise English. Return only English, nothing else."
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
