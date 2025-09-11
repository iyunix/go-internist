// G:\go_internist\internal\services\chat\sources.go
package chat

import (
    "strings"
    "github.com/pinecone-io/go-pinecone/v4/pinecone"
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

// ExtractSources extracts unique document titles from Pinecone matches
func (s *SourceExtractor) ExtractSources(matches []*pinecone.ScoredVector) []string {
    var sources []string
    seen := make(map[string]bool)
    
    s.logger.Info("extracting sources", "matches_count", len(matches))

    for _, match := range matches {
        if match == nil || match.Vector == nil || match.Vector.Metadata == nil {
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

func (s *SourceExtractor) extractTitle(match *pinecone.ScoredVector) string {
    md := match.Vector.Metadata.GetFields()
    
    var title string
    
    // Priority: source_file > section_heading > chunk ID
    if f, ok := md["source_file"]; ok {
        title = strings.TrimSpace(f.GetStringValue())
        title = s.cleanFilename(title)
    }
    
    if title == "" {
        if f, ok := md["section_heading"]; ok {
            title = strings.TrimSpace(f.GetStringValue())
        }
    }
    
    if title == "" {
        title = match.Vector.Id
    }
    
    return title
}

func (s *SourceExtractor) cleanFilename(filename string) string {
    // Clean up the filename - remove extension and path
    title := strings.TrimSuffix(filename, ".md")
    title = strings.TrimSuffix(title, "_Drug_information")
    title = strings.ReplaceAll(title, "_", " ")
    return title
}
