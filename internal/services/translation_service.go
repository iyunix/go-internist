// File: internal/services/translation_service.go
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
    client  *http.Client
    logger  Logger
}

type TranslationRequest struct {
    Model string `json:"model"`
    Input string `json:"input"`
}

// NEW: Correct response structure based on your test results
type TranslationResponse struct {
    Output []struct {
        Content []struct {
            Type string `json:"type"`
            Text string `json:"text"`
        } `json:"content"`
    } `json:"output"`
    Error *struct {
        Message string `json:"message"`
        Type    string `json:"type"`
    } `json:"error,omitempty"`
}

func NewTranslationService(apiKey string, logger Logger) *TranslationService {
    return &TranslationService{
        apiKey:  apiKey,
        baseURL: "https://api.avalai.ir/v1",
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

// FIXED: Correct response parsing
func (ts *TranslationService) TranslateToEnglish(ctx context.Context, text string) (string, error) {
    ts.logger.Info("Starting translation", "text", text)
    
    prompt := fmt.Sprintf(`Translate this Persian medical text to clear English. Return only the English translation: %s`, text)

    reqBody := TranslationRequest{
        Model: "gemini-2.5-flash-lite", // Using the working model from your test
        Input: prompt,
    }

    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        ts.logger.Error("Failed to marshal translation request", "error", err)
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", ts.baseURL+"/responses", bytes.NewBuffer(jsonData))
    if err != nil {
        ts.logger.Error("Failed to create translation request", "error", err)
        return "", fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+ts.apiKey)

    resp, err := ts.client.Do(req)
    if err != nil {
        ts.logger.Error("Translation API call failed", "error", err)
        return "", fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    ts.logger.Info("Translation API response received", "status_code", resp.StatusCode)

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        ts.logger.Error("Translation API returned error", "status_code", resp.StatusCode, "body", string(bodyBytes))
        return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
    }

    var translationResp TranslationResponse
    if err := json.NewDecoder(resp.Body).Decode(&translationResp); err != nil {
        ts.logger.Error("Failed to decode translation response", "error", err)
        return "", fmt.Errorf("failed to decode response: %w", err)
    }

    if translationResp.Error != nil {
        ts.logger.Error("Translation API error", "error_message", translationResp.Error.Message, "error_type", translationResp.Error.Type)
        return "", fmt.Errorf("translation API error: %s", translationResp.Error.Message)
    }

    // NEW: Correct parsing of nested response structure
    if len(translationResp.Output) == 0 {
        ts.logger.Error("No output in translation response")
        return "", fmt.Errorf("no output in translation response")
    }

    if len(translationResp.Output[0].Content) == 0 {
        ts.logger.Error("No content in translation response")
        return "", fmt.Errorf("no content in translation response")
    }

    // Extract the text from the nested structure
    translation := strings.TrimSpace(translationResp.Output[0].Content[0].Text)
    translation = strings.Trim(translation, "\"'")
    
    if translation == "" {
        ts.logger.Warn("Translation API returned empty result", "original", text)
        return "", fmt.Errorf("translation returned empty result")
    }
    
    ts.logger.Info("Translation completed successfully", "original", text, "translated", translation)
    
    return translation, nil
}
