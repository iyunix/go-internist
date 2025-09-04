package services

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/pinecone-io/go-pinecone/v4/pinecone"
    "google.golang.org/protobuf/types/known/structpb"
)

// PineconeService wraps the Pinecone client and config.
type PineconeService struct {
    client     *pinecone.Client
    indexHost  string
    namespace  string
    timeout    time.Duration
    maxRetries int
}

// NewPineconeService constructs a new service with default timeout and retry settings.
func NewPineconeService(apiKey, indexHost, namespace string) (*PineconeService, error) {
    pc, err := pinecone.NewClient(pinecone.NewClientParams{
        ApiKey: apiKey,
    })
    if err != nil {
        return nil, err
    }
    return &PineconeService{
        client:     pc,
        indexHost:  indexHost,
        namespace:  namespace,
        timeout:    8 * time.Second,
        maxRetries: 3,
    }, nil
}

// indexConn returns a connection to the Pinecone index, already using configured host and namespace.
func (s *PineconeService) indexConn() (*pinecone.IndexConnection, error) {
    return s.client.Index(pinecone.NewIndexConnParams{
        Host:      s.indexHost,
        Namespace: s.namespace,
    })
}

// retryWithTimeout runs a function with retries and a per-attempt timeout.
func (s *PineconeService) retryWithTimeout(call func(ctx context.Context) error) error {
    for attempt := 1; attempt <= s.maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
        err := call(ctx)
        cancel()
        if err == nil {
            return nil
        }
        log.Printf("[PineconeService] attempt %d/%d failed: %v", attempt, s.maxRetries, err)
        time.Sleep(time.Duration(attempt) * time.Second)
    }
    return errors.New("pinecone: operation failed after retries")
}
func (s *PineconeService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]any) error {
    return s.retryWithTimeout(func(ctx context.Context) error {
        idx, err := s.indexConn()
        if err != nil {
            return err
        }

        metadataStruct, err := structpb.NewStruct(metadata)
        if err != nil {
            return err
        }

        vectors := []*pinecone.Vector{
            {
                Id:       id,
                Values:   &values,         
                Metadata: metadataStruct,
            },
        }

        _, err = idx.UpsertVectors(ctx, vectors)
        return err
    })
}


// QuerySimilar returns the top K most similar vectors to the given embedding.
func (s *PineconeService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*pinecone.ScoredVector, error) {
    var result []*pinecone.ScoredVector
    err := s.retryWithTimeout(func(ctx context.Context) error {
        idx, err := s.indexConn()
        if err != nil {
            return err
        }

        resp, err := idx.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
            Vector:        embedding,
            TopK:          uint32(topK),
            IncludeValues: false,
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
