// G:\go_internist\internal\services\pinecone\client.go
package pinecone

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "github.com/qdrant/go-client/qdrant"
)

// ClientService implements Qdrant operations using direct HTTP REST API calls
type ClientService struct {
    config *Config
    client *http.Client
    baseURL string
    logger Logger
}

// NewClientService creates a new HTTP-based Qdrant client
func NewClientService(config *Config, logger Logger) (*ClientService, error) {
    if err := config.Validate(); err != nil {
        return nil, NewConfigError(err.Error())
    }
    
    // Build base URL for Qdrant Cloud REST API
    baseURL := fmt.Sprintf("https://%s", config.IndexHost)
    
    service := &ClientService{
        config: config,
        client: &http.Client{
            Timeout: 60 * time.Second,
        },
        baseURL: baseURL,
        logger: logger,
    }
    
    // Skip health check for now - test during first query
    logger.Info("Qdrant HTTP client initialized successfully",
        "url", baseURL,
        "collection", config.Namespace)
    
    return service, nil
}

func (c *ClientService) Client() *ClientService {
    return c
}

func (c *ClientService) IndexConnection() (*ClientService, error) {
    return c, nil
}

func (c *ClientService) createAuthContext(ctx context.Context) context.Context {
    return ctx
}

// HealthCheck tests the connection by getting collection info
func (c *ClientService) HealthCheck(ctx context.Context) error {
    url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.config.Namespace)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("Api-Key", c.config.APIKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        c.logger.Error("HTTP health check failed", "error", err)
        return NewConnectionError("health_check", "HTTP request failed", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        c.logger.Error("HTTP health check failed", "status", resp.StatusCode, "body", string(body))
        return NewConnectionError("health_check", fmt.Sprintf("HTTP %d", resp.StatusCode), fmt.Errorf(string(body)))
    }
    
    c.logger.Debug("Qdrant HTTP health check passed")
    return nil
}

// Query performs vector search using Qdrant HTTP REST API
func (c *ClientService) Query(ctx context.Context, req *QueryRequest) ([]*qdrant.ScoredPoint, error) {
    url := fmt.Sprintf("%s/collections/%s/points/query", c.baseURL, c.config.Namespace)
    
    // Create request body matching Qdrant REST API
    requestBody := map[string]interface{}{
        "query":        req.Query,
        "limit":        req.Limit,
        "with_payload": true,
    }
    
    bodyBytes, err := json.Marshal(requestBody)
    if err != nil {
        return nil, err
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Api-Key", c.config.APIKey)
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
    }
    
    // Parse response
    var response QueryResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }
    
    // Convert HTTP response to qdrant.ScoredPoint format
    scoredPoints := make([]*qdrant.ScoredPoint, len(response.Result.Points))
    for i, point := range response.Result.Points {
        scoredPoints[i] = &qdrant.ScoredPoint{
            Id: &qdrant.PointId{
                PointIdOptions: &qdrant.PointId_Uuid{Uuid: point.ID},
            },
            Score: point.Score,
            Payload: convertPayloadFromHTTP(point.Payload),
        }
    }
    
    return scoredPoints, nil
}

// Helper types for HTTP API
type QueryRequest struct {
    Query []float32
    Limit uint64
}

type QueryResponse struct {
    Result struct {
        Points []struct {
            ID      string                 `json:"id"`
            Score   float32               `json:"score"`
            Payload map[string]interface{} `json:"payload"`
        } `json:"points"`
    } `json:"result"`
}

// Convert HTTP payload to qdrant.Value format
func convertPayloadFromHTTP(httpPayload map[string]interface{}) map[string]*qdrant.Value {
    payload := make(map[string]*qdrant.Value)
    for k, v := range httpPayload {
        switch val := v.(type) {
        case string:
            payload[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
        case float64:
            payload[k] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: val}}
        case bool:
            payload[k] = &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: val}}
        default:
            payload[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
        }
    }
    return payload
}

func (c *ClientService) GetStatus(ctx context.Context) ServiceStatus {
    err := c.HealthCheck(ctx)
    isHealthy := err == nil
    
    return ServiceStatus{
        IsHealthy:         isHealthy,
        ConnectionHealthy: isHealthy,
        IndexHealthy:      isHealthy,
        IndexHost:         c.config.IndexHost,
        Namespace:         c.config.Namespace,
        Message:          "Qdrant HTTP service",
    }
}

func (c *ClientService) Close() error {
    return nil // HTTP client doesn't need explicit closing
}
