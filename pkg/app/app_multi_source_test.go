package app

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/formatters"
	"github.com/roverdotcom/snagsby/pkg/resolvers"
)

// mockResolver is a test resolver that simulates AWS resolvers
type mockResolver struct {
	items map[string]string
	err   error
}

func (m *mockResolver) Resolve(source *config.Source) *resolvers.Result {
	result := &resolvers.Result{Source: source}
	if m.err != nil {
		result.AppendError(m.err)
		return result
	}
	result.AppendItems(m.items)
	return result
}

// TestMultiSourceIntegrationWithMocks tests the integration of file://, sm://, and s3://
// sources using mocks for AWS services
func TestMultiSourceIntegrationWithMocks(t *testing.T) {
	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "base.env")

	envContent := `# Base configuration
DATABASE_HOST=localhost
DATABASE_PORT=5432
API_KEY=file_api_key
SHARED_SECRET=from_file
`

	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Simulate the full resolution flow with multiple source types
	// In a real scenario, these would be:
	// - file://./base.env
	// - sm://production/database/*
	// - s3://my-bucket/config.json

	// 1. Resolve file:// source (real resolver)
	fileURL, _ := url.Parse("file://" + envFile)
	fileSource := &config.Source{URL: fileURL}
	fileResolver := &resolvers.FileResolver{}
	fileResult := fileResolver.Resolve(fileSource)

	if fileResult.HasErrors() {
		t.Fatalf("File resolver failed: %v", fileResult.Errors)
	}

	// 2. Simulate sm:// source (mock)
	smURL, _ := url.Parse("sm://production/database/*")
	smSource := &config.Source{URL: smURL}
	mockSMResolver := &mockResolver{
		items: map[string]string{
			"DATABASE_PASSWORD": "secret_from_sm",
			"DATABASE_USER":     "admin",
			"SHARED_SECRET":     "from_sm", // This should overwrite file://
		},
	}
	smResult := mockSMResolver.Resolve(smSource)

	// 3. Simulate s3:// source (mock)
	s3URL, _ := url.Parse("s3://my-bucket/config.json")
	s3Source := &config.Source{URL: s3URL}
	mockS3Resolver := &mockResolver{
		items: map[string]string{
			"CACHE_ENDPOINT": "redis.example.com",
			"CACHE_PORT":     "6379",
			"API_KEY":        "s3_api_key",      // This should overwrite file://
			"SHARED_SECRET":  "from_s3_final",  // This should overwrite both file:// and sm://
		},
	}
	s3Result := mockS3Resolver.Resolve(s3Source)

	// 4. Merge results in order (simulating what the app does)
	results := []*resolvers.Result{fileResult, smResult, s3Result}
	var itemMaps []map[string]string
	for _, result := range results {
		itemMaps = append(itemMaps, result.Items)
	}

	merged := formatters.Merge(itemMaps)

	// 5. Verify the final merged result
	// Later sources should overwrite earlier ones
	expected := map[string]string{
		// From file:// (not overwritten)
		"DATABASE_HOST": "localhost",
		"DATABASE_PORT": "5432",

		// From sm:// (overwrites nothing, adds new)
		"DATABASE_PASSWORD": "secret_from_sm",
		"DATABASE_USER":     "admin",

		// From s3:// (overwrites file://)
		"CACHE_ENDPOINT": "redis.example.com",
		"CACHE_PORT":     "6379",

		// Overwrite chain: file:// -> sm:// -> s3://
		"API_KEY":       "s3_api_key",      // s3 wins over file
		"SHARED_SECRET": "from_s3_final",   // s3 wins over sm and file
	}

	if len(merged) != len(expected) {
		t.Errorf("Expected %d merged items, got %d", len(expected), len(merged))
		t.Logf("Merged items: %+v", merged)
	}

	for key, expectedValue := range expected {
		if actualValue, ok := merged[key]; !ok {
			t.Errorf("Expected key %s not found in merged results", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// 6. Verify overwrite behavior specifically
	t.Run("verify_overwrite_chain", func(t *testing.T) {
		// SHARED_SECRET should be overwritten in this order:
		// file:// "from_file" -> sm:// "from_sm" -> s3:// "from_s3_final"

		if fileResult.Items["SHARED_SECRET"] != "from_file" {
			t.Errorf("File result should have SHARED_SECRET=from_file")
		}

		if smResult.Items["SHARED_SECRET"] != "from_sm" {
			t.Errorf("SM result should have SHARED_SECRET=from_sm")
		}

		if s3Result.Items["SHARED_SECRET"] != "from_s3_final" {
			t.Errorf("S3 result should have SHARED_SECRET=from_s3_final")
		}

		if merged["SHARED_SECRET"] != "from_s3_final" {
			t.Errorf("Merged result should have SHARED_SECRET=from_s3_final (last source wins)")
		}
	})
}

// TestRealWorldScenario simulates a real-world usage pattern
func TestRealWorldScenario(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env.defaults (committed to repo)
	defaultsFile := filepath.Join(tmpDir, ".env.defaults")
	defaultsContent := `# Default configuration values
APP_NAME=myapp
APP_ENV=development
LOG_LEVEL=info
DATABASE_HOST=localhost
DATABASE_PORT=5432
CACHE_TTL=300
`
	err := os.WriteFile(defaultsFile, []byte(defaultsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create defaults file: %v", err)
	}

	// Create .env.local (local overrides, not committed)
	localFile := filepath.Join(tmpDir, ".env.local")
	localContent := `# Local development overrides
LOG_LEVEL=debug
DATABASE_HOST=127.0.0.1
`
	err = os.WriteFile(localFile, []byte(localContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}

	// Resolve defaults
	defaultsURL, _ := url.Parse("file://" + defaultsFile)
	defaultsSource := &config.Source{URL: defaultsURL}
	defaultsResolver := &resolvers.FileResolver{}
	defaultsResult := defaultsResolver.Resolve(defaultsSource)

	// Resolve local overrides
	localURL, _ := url.Parse("file://" + localFile)
	localSource := &config.Source{URL: localURL}
	localResolver := &resolvers.FileResolver{}
	localResult := localResolver.Resolve(localSource)

	// Mock AWS Secrets Manager (would fetch DATABASE_PASSWORD)
	smURL, _ := url.Parse("sm://production/api/*")
	smSource := &config.Source{URL: smURL}
	mockSMResolver := &mockResolver{
		items: map[string]string{
			"API_KEY":    "prod_api_key_from_aws",
			"API_SECRET": "prod_secret_from_aws",
		},
	}
	smResult := mockSMResolver.Resolve(smSource)

	// Merge: defaults < local < aws
	merged := formatters.Merge([]map[string]string{
		defaultsResult.Items,
		localResult.Items,
		smResult.Items,
	})

	// Verify expected behavior
	tests := []struct {
		key      string
		expected string
		source   string
	}{
		{"APP_NAME", "myapp", "defaults (not overridden)"},
		{"APP_ENV", "development", "defaults (not overridden)"},
		{"LOG_LEVEL", "debug", "local override wins"},
		{"DATABASE_HOST", "127.0.0.1", "local override wins"},
		{"DATABASE_PORT", "5432", "defaults (not overridden)"},
		{"CACHE_TTL", "300", "defaults (not overridden)"},
		{"API_KEY", "prod_api_key_from_aws", "from AWS SM"},
		{"API_SECRET", "prod_secret_from_aws", "from AWS SM"},
	}

	for _, tt := range tests {
		if actual, ok := merged[tt.key]; !ok {
			t.Errorf("Key %s not found (expected from %s)", tt.key, tt.source)
		} else if actual != tt.expected {
			t.Errorf("Key %s: expected %q (from %s), got %q", tt.key, tt.expected, tt.source, actual)
		}
	}
}

// TestSourceOrderMatters verifies that the order of sources determines precedence
func TestSourceOrderMatters(t *testing.T) {
	tmpDir := t.TempDir()

	// Source 1
	file1 := filepath.Join(tmpDir, "first.env")
	os.WriteFile(file1, []byte("KEY=first\n"), 0644)

	// Source 2
	file2 := filepath.Join(tmpDir, "second.env")
	os.WriteFile(file2, []byte("KEY=second\n"), 0644)

	// Source 3
	file3 := filepath.Join(tmpDir, "third.env")
	os.WriteFile(file3, []byte("KEY=third\n"), 0644)

	// Test order: 1 -> 2 -> 3 (3 should win)
	cfg := config.NewConfig()
	url1, _ := url.Parse("file://" + file1)
	url2, _ := url.Parse("file://" + file2)
	url3, _ := url.Parse("file://" + file3)

	cfg.Sources = append(cfg.Sources, &config.Source{URL: url1})
	cfg.Sources = append(cfg.Sources, &config.Source{URL: url2})
	cfg.Sources = append(cfg.Sources, &config.Source{URL: url3})

	results := ResolveConfigSources(cfg)

	var itemMaps []map[string]string
	for _, result := range results {
		itemMaps = append(itemMaps, result.Items)
	}

	merged := formatters.Merge(itemMaps)

	if merged["KEY"] != "third" {
		t.Errorf("Expected last source to win: got %q, wanted %q", merged["KEY"], "third")
	}
}
