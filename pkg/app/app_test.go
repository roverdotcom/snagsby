package app

import (
	"net/url"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
)

func TestResolveConfigSources(t *testing.T) {
	// Test with empty config
	emptyConfig := config.NewConfig()
	results := ResolveConfigSources(emptyConfig)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty config, got %d", len(results))
	}

	// Test with invalid scheme - should return error
	invalidConfig := config.NewConfig()
	invalidURL, _ := url.Parse("invalid://test/path")
	invalidSource := &config.Source{URL: invalidURL}
	invalidConfig.Sources = []*config.Source{invalidSource}

	results = ResolveConfigSources(invalidConfig)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if !results[0].HasErrors() {
		t.Errorf("Expected error for invalid scheme")
	}

	// Test with multiple invalid sources to verify parallel processing
	multiConfig := config.NewConfig()
	url1, _ := url.Parse("invalid://test/one")
	url2, _ := url.Parse("invalid://test/two")
	url3, _ := url.Parse("invalid://test/three")
	multiConfig.Sources = []*config.Source{
		{URL: url1},
		{URL: url2},
		{URL: url3},
	}

	results = ResolveConfigSources(multiConfig)
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	for i, result := range results {
		if !result.HasErrors() {
			t.Errorf("Expected error for result %d", i)
		}
	}
}
