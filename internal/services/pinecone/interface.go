// G:\go_internist\internal\services\pinecone\interface.go
package pinecone

import (
    "context"
    "github.com/qdrant/go-client/qdrant"
)

// ClientProvider handles Qdrant client and connection management
type ClientProvider interface {
    Client() *qdrant.Client
    HealthCheck(ctx context.Context) error
}

// VectorRepository handles vector data operations
type VectorRepository interface {
    UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]interface{}) error
    QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*qdrant.ScoredPoint, error)
    DeleteVector(ctx context.Context, id string) error
    FetchVector(ctx context.Context, id string) (*qdrant.RetrievedPoint, error)
}

// RetryProvider handles retry logic for Qdrant operations
type RetryProvider interface {
    RetryWithTimeout(call func(ctx context.Context) error) error
}

// Service combines all Qdrant capabilities
type Service interface {
    ClientProvider
    VectorRepository
    RetryProvider
    GetStatus(ctx context.Context) ServiceStatus
}

// ServiceStatus represents Qdrant service health
type ServiceStatus struct {
    IsHealthy         bool
    ConnectionHealthy bool
    IndexHealthy      bool  // Keep same field name for compatibility
    Message           string
    IndexHost         string // Keep same field name for compatibility
    Namespace         string // Keep same field name for compatibility
}

// Logger interface for Qdrant operations
type Logger interface {
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}
