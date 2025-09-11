// G:\go_internist\internal\services\pinecone_service.go
package services

import (
    "context"
    "github.com/iyunix/go-internist/internal/services/pinecone"
    pineconeSDK "github.com/pinecone-io/go-pinecone/v4/pinecone"  // ADD ALIAS
)

type PineconeService struct {
    config        *pinecone.Config
    clientService *pinecone.ClientService
    retryService  *pinecone.RetryService
    vectorService *pinecone.VectorService
    logger        Logger
}

func NewPineconeService(apiKey, indexHost, namespace string) (*PineconeService, error) {
    // Create configuration with defaults
    config := pinecone.DefaultConfig()
    config.APIKey = apiKey
    config.IndexHost = indexHost
    config.Namespace = namespace
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, pinecone.NewConfigError(err.Error())
    }
    
    // Create logger
    logger := &NoOpLogger{} // Will be replaced with production logger
    
    // Create modular components
    clientService, err := pinecone.NewClientService(config, logger)
    if err != nil {
        return nil, err
    }
    
    retryService := pinecone.NewRetryService(config, logger)
    vectorService := pinecone.NewVectorService(clientService, retryService, config, logger)
    
    return &PineconeService{
        config:        config,
        clientService: clientService,
        retryService:  retryService,
        vectorService: vectorService,
        logger:        logger,
    }, nil
}

// Vector Operations - UPDATE TYPE REFERENCES
func (s *PineconeService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]any) error {
    return s.vectorService.UpsertVector(ctx, id, values, metadata)
}

func (s *PineconeService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*pineconeSDK.ScoredVector, error) {
    return s.vectorService.QuerySimilar(ctx, embedding, topK)
}

func (s *PineconeService) DeleteVector(ctx context.Context, id string) error {
    return s.vectorService.DeleteVector(ctx, id)
}

func (s *PineconeService) FetchVector(ctx context.Context, id string) (*pineconeSDK.Vector, error) {
    return s.vectorService.FetchVector(ctx, id)
}

// Service Management
func (s *PineconeService) HealthCheck(ctx context.Context) error {
    return s.clientService.HealthCheck(ctx)
}

func (s *PineconeService) GetStatus(ctx context.Context) pinecone.ServiceStatus {
    return s.clientService.GetStatus(ctx)
}

// Retry Operations
func (s *PineconeService) RetryWithTimeout(call func(ctx context.Context) error) error {
    return s.retryService.RetryWithTimeout(call)
}
