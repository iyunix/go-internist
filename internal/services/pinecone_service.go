// File: internal/services/pinecone_service.go
package services

import (
    "context"
    "errors"
    "log"
    "time"

    pinecone "github.com/pinecone-io/go-pinecone/pinecone"
)

type PineconeService struct {
    client     *pinecone.Client
    indexName  string
    namespace  string
    timeout    time.Duration // For API call timeouts
    maxRetries int           // For retry logic
}

func NewPineconeService(apiKey, indexName, namespace string) *PineconeService {
    client := pinecone.NewClient(apiKey)
    return &PineconeService{
        client:     client,
        indexName:  indexName,
        namespace:  namespace,
        timeout:    8 * time.Second,
        maxRetries: 3,
    }
}

// retryWithTimeout wraps Pinecone API calls
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

// UpsertVector upserts a vector embedding with retry and timeout
func (s *PineconeService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]string) error {
    return s.retryWithTimeout(func(ctx context.Context) error {
        upsertReq := pinecone.UpsertRequest{
            IndexName: s.indexName,
            Namespace: s.namespace,
            Vectors: []pinecone.Vector{
                {
                    ID:       id,
                    Values:   values,
                    Metadata: metadata,
                },
            },
        }
        _, err := s.client.Upsert(ctx, upsertReq)
        return err
    })
}

// QuerySimilar queries for similar documents with reliability
func (s *PineconeService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]pinecone.Match, error) {
    var result []pinecone.Match
    err := s.retryWithTimeout(func(ctx context.Context) error {
        queryReq := pinecone.QueryRequest{
            IndexName: s.indexName,
            Namespace: s.namespace,
            TopK:      topK,
            Values:    embedding,
        }
        resp, err := s.client.Query(ctx, queryReq)
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
