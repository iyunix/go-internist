// G:\go_internist\internal\services\chat\sources.go
package chat

import (
    "strings"
    "github.com/qdrant/go-client/qdrant"
)

type SourceExtractor struct {
    config *Config
    logger Logger
}

func NewSourceExtractor(config *Config, logger Logger) *SourceExtractor {
    return &SourceExtractor{
        config: config,
        logger: logger,
    }
}

// ExtractSources extracts unique document titles from Qdrant matches
func (s *SourceExtractor) ExtractSources(matches []*qdrant.ScoredPoint) []string {
    var sources []string
    seen := make(map[string]bool)
    
    s.logger.Info("extracting sources", "matches_count", len(matches))

    for _, match := range matches {
        if match == nil || match.Payload == nil {
            continue
        }
        
        title := s.extractTitle(match)
        
        // Add unique titles only
        if title != "" && !seen[title] {
            sources = append(sources, title)
            seen[title] = true
            
            // Limit sources based on config
            if len(sources) >= s.config.MaxSources {
                break
            }
        }
    }
    
    s.logger.Info("sources extracted", "unique_sources", len(sources))
    return sources
}

func (s *SourceExtractor) extractTitle(match *qdrant.ScoredPoint) string {
    var title string
    
    // Priority: source_file > section_heading > chunk ID
    if sourceFile := s.getStringFromQdrantPayload(match.Payload, "source_file"); sourceFile != "" {
        title = strings.TrimSpace(sourceFile)
        title = s.cleanFilename(title)
    }
    
    if title == "" {
        if sectionHeading := s.getStringFromQdrantPayload(match.Payload, "section_heading"); sectionHeading != "" {
            title = strings.TrimSpace(sectionHeading)
        }
    }
    
    if title == "" {
        // Extract point ID from Qdrant ScoredPoint
        title = s.extractPointID(match)
    }
    
    return title
}

// getStringFromQdrantPayload safely extracts string values from Qdrant payload
func (s *SourceExtractor) getStringFromQdrantPayload(payload map[string]*qdrant.Value, key string) string {
    if payload == nil {
        return ""
    }
    
    value, ok := payload[key]
    if !ok || value == nil {
        return ""
    }
    
    switch v := value.Kind.(type) {
    case *qdrant.Value_StringValue:
        return v.StringValue
    default:
        return ""
    }
}

// extractPointID safely extracts the point ID from a Qdrant ScoredPoint
func (s *SourceExtractor) extractPointID(point *qdrant.ScoredPoint) string {
    if point.Id == nil {
        return ""
    }
    
    switch id := point.Id.PointIdOptions.(type) {
    case *qdrant.PointId_Num:
        return string(rune(id.Num))
    case *qdrant.PointId_Uuid:
        return id.Uuid
    default:
        return ""
    }
}

func (s *SourceExtractor) cleanFilename(filename string) string {
    // Clean up the filename - remove extension and path
    title := strings.TrimSuffix(filename, ".md")
    title = strings.TrimSuffix(title, "_Drug_information")
    title = strings.ReplaceAll(title, "_", " ")
    return title
}
