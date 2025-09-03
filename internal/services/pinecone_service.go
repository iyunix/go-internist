// File: internal/services/pinecone_service.go
package services

import (
	"context"
	"errors"
	"log"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type PineconeService struct {
	index     *pinecone.IndexConnection
	namespace string
}

// NewPineconeService initializes the connection to the Pinecone index.
func NewPineconeService(apiKey, indexName, namespace string) (*PineconeService, error) {
	pc, err := pinecone.NewClient(pinecone.NewClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}

	idx, err := pc.Index(indexName)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Pinecone client initialized successfully.")
	return &PineconeService{index: idx, namespace: namespace}, nil
}

// Query finds the most relevant documents for a given vector.
func (s *PineconeService) Query(ctx context.Context, vector []float32) (string, error) {
	queryResult, err := s.index.Query(ctx, &pinecone.QueryRequest{
		Vector:          vector,
		TopK:            15,
		IncludeMetadata: true,
		Namespace:       s.namespace,
	})
	if err != nil {
		return "", err
	}

	if len(queryResult.Matches) == 0 {
		return "", errors.New("no relevant documents found in Pinecone")
	}

	// Combine the text from the retrieved documents to create the context.
	var contextText string
	for _, match := range queryResult.Matches {
		if text, ok := match.Metadata["text"].(string); ok {
			contextText += text + "\n\n"
		}
	}

	return contextText, nil
}