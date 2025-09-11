package pinecone

import (
    "context"
    "fmt"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
    "google.golang.org/protobuf/types/known/structpb"
)

type VectorService struct {
    clientService *ClientService
    retryService  *RetryService
    config        *Config
    logger        Logger
}

func NewVectorService(clientService *ClientService, retryService *RetryService, config *Config, logger Logger) *VectorService {
    return &VectorService{
        clientService: clientService,
        retryService:  retryService,
        config:        config,
        logger:        logger,
    }
}

func (v *VectorService) UpsertVector(ctx context.Context, id string, values []float32, metadata map[string]any) error {
    v.logger.Info("upserting vector", 
        "vector_id", id,
        "dimensions", len(values),
        "metadata_fields", len(metadata))
    
    return v.retryService.RetryWithTimeout(func(ctx context.Context) error {
        idx, err := v.clientService.IndexConnection()
        if err != nil {
            return err
        }
        
        metadataStruct, err := structpb.NewStruct(metadata)
        if err != nil {
            return NewVectorError("upsert", id, "failed to convert metadata", err)
        }
        
        vectors := []*pinecone.Vector{
            {
                Id:       id,
                Values:   &values,
                Metadata: metadataStruct,
            },
        }
        
        _, err = idx.UpsertVectors(ctx, vectors)
        if err != nil {
            return NewVectorError("upsert", id, "failed to upsert vector", err)
        }
        
        return nil
    })
}

func (v *VectorService) QuerySimilar(ctx context.Context, embedding []float32, topK int) ([]*pinecone.ScoredVector, error) {
    v.logger.Info("querying similar vectors", 
        "embedding_dimensions", len(embedding),
        "top_k", topK)
    
    // Validate topK against config limits
    if topK > v.config.TopKLimit {
        return nil, NewQueryError("query_similar", 
            fmt.Sprintf("topK %d exceeds limit %d", topK, v.config.TopKLimit), nil)
    }
    
    var result []*pinecone.ScoredVector
    
    err := v.retryService.RetryWithTimeout(func(ctx context.Context) error {
        idx, err := v.clientService.IndexConnection()
        if err != nil {
            return err
        }
        
        resp, err := idx.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
            Vector:          embedding,
            TopK:            uint32(topK),
            IncludeValues:   v.config.IncludeValues,
            IncludeMetadata: true,
        })
        if err != nil {
            return NewQueryError("query_similar", "failed to query vectors", err)
        }
        
        result = resp.Matches
        return nil
    })
    
    if err != nil {
        v.logger.Error("vector query failed", "error", err, "top_k", topK)
        return nil, err
    }
    
    v.logger.Info("vector query completed", 
        "results_count", len(result),
        "top_k", topK)
    
    return result, nil
}

func (v *VectorService) DeleteVector(ctx context.Context, id string) error {
    v.logger.Info("deleting vector", "vector_id", id)
    
    return v.retryService.RetryWithTimeout(func(ctx context.Context) error {
        idx, err := v.clientService.IndexConnection()
        if err != nil {
            return err
        }
        
        err = idx.DeleteVectorsById(ctx, []string{id})
        if err != nil {
            return NewVectorError("delete", id, "failed to delete vector", err)
        }
        
        return nil
    })
}

func (v *VectorService) FetchVector(ctx context.Context, id string) (*pinecone.Vector, error) {
    v.logger.Info("fetching vector", "vector_id", id)
    
    var result *pinecone.Vector
    
    err := v.retryService.RetryWithTimeout(func(ctx context.Context) error {
        idx, err := v.clientService.IndexConnection()
        if err != nil {
            return err
        }
        
        // CORRECTED: Use proper Pinecone FetchVectors API
        resp, err := idx.FetchVectors(ctx, []string{id})
        if err != nil {
            return NewVectorError("fetch", id, "failed to fetch vector", err)
        }
        
        if vector, exists := resp.Vectors[id]; exists {
            result = vector
        } else {
            return NewVectorError("fetch", id, "vector not found", nil)
        }
        
        return nil
    })
    
    return result, err
}
