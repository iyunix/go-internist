// G:\go_internist\internal\services\pinecone_service.go
package services

import (
    "context"
    "github.com/iyunix/go-internist/internal/services/pinecone"
    qdrantSDK "github.com/qdrant/go-client/qdrant"
)

type PineconeService struct {
    config        *pinecone.Config
    clientService *pinecone.ClientService
    retryService  *pinecone.RetryService
    vectorService *pinecone.VectorService
    logger        Logger
}

func NewPineconeService(apiKey, indexHost, namespace string, logger Logger) (*PineconeService, error) {
    config := pinecone.DefaultConfig()
    config.APIKey = apiKey
    config.IndexHost = indexHost  // This will be Qdrant URL
    config.Namespace = namespace  // This will be Qdrant collection name

    if err := config.Validate(); err != nil {
        return nil, pinecone.NewConfigError(err.Error())
    }

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

// Vector Operations - Updated to return Qdrant types
func (s *PineconeService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]any) error {
    return s.vectorService.UpsertVector(ctx, id, values, metadata)
}

func (s *PineconeService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*qdrantSDK.ScoredPoint, error) {
    return s.vectorService.QuerySimilar(ctx, embedding, topK)
}

func (s *PineconeService) DeleteVector(ctx context.Context, id string) error {
    return s.vectorService.DeleteVector(ctx, id)
}

func (s *PineconeService) FetchVector(ctx context.Context, id string) (*qdrantSDK.RetrievedPoint, error) {
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
func (s *PineconeService) RetryWithTimeout(parentCtx context.Context, call func(ctx context.Context) error) error {
    return s.retryService.RetryWithTimeout(call)
}

// Cleanup
func (s *PineconeService) Close() error {
    return s.clientService.Close()
}
