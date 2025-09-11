// G:\go_internist\internal\services\pinecone\interface.go
package pinecone

import (
    "context"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
)

// ClientProvider handles Pinecone client and connection management
type ClientProvider interface {
    IndexConnection() (*pinecone.IndexConnection, error)
    HealthCheck(ctx context.Context) error
}

// VectorRepository handles vector data operations
type VectorRepository interface {
    UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]any) error
    QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*pinecone.ScoredVector, error)
    DeleteVector(ctx context.Context, id string) error
    FetchVector(ctx context.Context, id string) (*pinecone.Vector, error)
}

// RetryProvider handles retry logic for Pinecone operations
type RetryProvider interface {
    RetryWithTimeout(call func(ctx context.Context) error) error
}

// Service combines all Pinecone capabilities
type Service interface {
    ClientProvider
    VectorRepository
    RetryProvider
    GetStatus(ctx context.Context) ServiceStatus
}

// ServiceStatus represents Pinecone service health
type ServiceStatus struct {
    IsHealthy         bool
    ConnectionHealthy bool
    IndexHealthy      bool
    Message           string
    IndexHost         string
    Namespace         string
}

// Logger interface for Pinecone operations
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}
