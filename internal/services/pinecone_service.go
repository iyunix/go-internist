// File: internal/services/pinecone_service.go
package services

import (
    "context"
    "errors"
    "log"
    "time"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type PineconeService struct {
    client     *pinecone.Client
    indexName  string
    namespace  string
    timeout    time.Duration
    maxRetries int
}

// Modern NewPineconeService with error handling for client construction
func NewPineconeService(apiKey, environment, indexName, namespace string) (*PineconeService, error) {
    client, err := pinecone.NewClient(pinecone.NewClientParams{
        ApiKey:      apiKey,
        Environment: environment,
    })
    if err != nil {
        return nil, err
    }
    return &PineconeService{
        client:     client,
        indexName:  indexName,
        namespace:  namespace,
        timeout:    8 * time.Second,
        maxRetries: 3,
    }, nil
}

func (s *PineconeService) retryWithTimeout(call func(ctx context.Context) error) error {
    for attempt := 1; attempt <= s.maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
        defer cancel()
        err := call(ctx)
        if err == nil {
            return nil
        }
        log.Printf("[PineconeService] Attempt %d/%d failed: %v", attempt, s.maxRetries, err)
        time.Sleep(time.Duration(attempt) * time.Second)
    }
    return errors.New("pinecone: operation failed after retries")
}

// UpsertVector upserts a vector embedding
func (s *PineconeService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]interface{}) error {
    return s.retryWithTimeout(func(ctx context.Context) error {
        upsertReq := &pinecone.UpsertRequest{
            Vectors: []*pinecone.Vector{
                {
                    Id:       id,
                    Values:   values,
                    Metadata: pinecone.NewMetadata(metadata),
                },
            },
            Namespace: s.namespace,
        }
        _, err := s.client.Upsert(ctx, s.indexName, upsertReq)
        return err
    })
}

// QuerySimilar queries for similar documents
func (s *PineconeService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*pinecone.Match, error) {
    var result []*pinecone.Match
    err := s.retryWithTimeout(func(ctx context.Context) error {
        queryReq := &pinecone.QueryRequest{
            Namespace: s.namespace,
            TopK:      uint32(topK),
            Vector:    embedding,
        }
        resp, err := s.client.Query(ctx, s.indexName, queryReq)
        if err != nil {
            return err
        }
        result = resp.Matches
        return nil
    })
    if err != nil {
        log.Printf("[PineconeService] QuerySimilar failed: %v", err)
        return nil, err
    }
    return result, nil
}
