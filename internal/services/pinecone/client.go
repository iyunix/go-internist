// G:\go_internist\internal\services\pinecone\client.go
package pinecone

import (
    "context"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type ClientService struct {
    config *Config
    client *pinecone.Client
    logger Logger
}

func NewClientService(config *Config, logger Logger) (*ClientService, error) {
    pc, err := pinecone.NewClient(pinecone.NewClientParams{
        ApiKey: config.APIKey,
    })
    if err != nil {
        return nil, NewConnectionError("client_init", "failed to create Pinecone client", err)
    }
    
    return &ClientService{
        config: config,
        client: pc,
        logger: logger,
    }, nil
}

func (c *ClientService) IndexConnection() (*pinecone.IndexConnection, error) {
    c.logger.Debug("creating index connection", 
        "index_host", c.config.IndexHost,
        "namespace", c.config.Namespace)
    
    conn, err := c.client.Index(pinecone.NewIndexConnParams{
        Host:      c.config.IndexHost,
        Namespace: c.config.Namespace,
    })
    if err != nil {
        c.logger.Error("index connection failed", "error", err,
            "index_host", c.config.IndexHost,
            "namespace", c.config.Namespace)
        return nil, NewConnectionError("index_connection", "failed to connect to index", err)
    }
    
    c.logger.Debug("index connection established successfully")
    return conn, nil
}

func (c *ClientService) HealthCheck(ctx context.Context) error {
    // Simple health check - attempt to create connection
    _, err := c.IndexConnection()
    if err != nil {
        return NewConnectionError("health_check", "index connection failed", err)
    }
    return nil
}

func (c *ClientService) GetStatus(ctx context.Context) ServiceStatus {
    err := c.HealthCheck(ctx)
    isHealthy := err == nil
    
    status := ServiceStatus{
        IsHealthy:         isHealthy,
        ConnectionHealthy: isHealthy,
        IndexHealthy:      isHealthy,
        IndexHost:         c.config.IndexHost,
        Namespace:         c.config.Namespace,
    }
    
    if err != nil {
        status.Message = err.Error()
    } else {
        status.Message = "Pinecone service healthy"
    }
    
    return status
}
