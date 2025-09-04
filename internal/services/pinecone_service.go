package services

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type PineconeService struct {
    client    *pinecone.Client
    indexName string
    namespace string
    timeout   time.Duration
    maxRetries int
}

// NewPineconeService constructs client with ApiKey only (no Environment field)
func NewPineconeService(apiKey, indexName, namespace string) (*PineconeService, error) {
    client, err := pinecone.NewClient(pinecone.NewClientParams{
        ApiKey: apiKey,
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

// UpsertVector upserts a vector embedding using the client’s Index method (Index().Vectors.Upsert)
func (s *PineconeService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]string) error {
    return s.retryWithTimeout(func(ctx context.Context) error {
        vectors := []pinecone.Vector{
            {
                Id:       id,
                Values:   values,
                Metadata: metadata,
            },
        }
        _, err := s.client.Index(s.indexName).Vectors.Upsert(ctx, pinecone.VectorsUpsertRequest{
            Vectors:   vectors,
            Namespace: s.namespace,
        })
        return err
    })
}

// QuerySimilar queries similar vectors
func (s *PineconeService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]pinecone.Match, error) {
    var result []pinecone.Match
    err := s.retryWithTimeout(func(ctx context.Context) error {
        resp, err := s.client.Index(s.indexName).Query(ctx, pinecone.QueryRequest{
            Vector:    embedding,
            TopK:      uint32(topK),
            Namespace: s.namespace,
        })
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
