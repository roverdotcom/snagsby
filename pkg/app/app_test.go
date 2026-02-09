package app

import (
	"net/url"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
)

func TestResolveConfigSources(t *testing.T) {
	tests := []struct {
		name           string
		sources        []string
		expectedCount  int
		expectErrors   bool
		checkAllErrors bool // if true, check that all results have errors
	}{
		{
			name:          "empty config",
			sources:       []string{},
			expectedCount: 0,
			expectErrors:  false,
		},
		{
			name:          "single invalid scheme",
			sources:       []string{"invalid://test/path"},
			expectedCount: 1,
			expectErrors:  true,
		},
		{
			name:           "multiple invalid sources for parallel processing",
			sources:        []string{"invalid://test/one", "invalid://test/two", "invalid://test/three"},
			expectedCount:  3,
			expectErrors:   true,
			checkAllErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewConfig()
			for _, sourceURL := range tt.sources {
				parsedURL, err := url.Parse(sourceURL)
				if err != nil {
					t.Fatalf("Failed to parse URL %s: %v", sourceURL, err)
				}
				cfg.Sources = append(cfg.Sources, &config.Source{URL: parsedURL})
			}

			results := ResolveConfigSources(cfg)

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectErrors {
				if tt.checkAllErrors {
					for i, result := range results {
						if !result.HasErrors() {
							t.Errorf("Expected error for result %d", i)
						}
					}
				} else if len(results) > 0 && !results[0].HasErrors() {
					t.Errorf("Expected error for first result")
				}
			}
		})
	}
}
