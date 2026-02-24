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

// TestMultiSourceOrderPreservation tests that ResolveConfigSources preserves
// source order, which is critical for the "later sources overwrite earlier ones" behavior.
// This test uses real file:// resolvers to ensure we're testing the actual production code path.
func TestMultiSourceOrderPreservation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create three .env files that all define the same keys with different values
	file1 := filepath.Join(tmpDir, "first.env")
	file1Content := `# First source
SHARED_KEY=from_first
ONLY_IN_FIRST=value1
DATABASE_HOST=localhost
`
	err := os.WriteFile(file1, []byte(file1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create first file: %v", err)
	}

	file2 := filepath.Join(tmpDir, "second.env")
	file2Content := `# Second source
SHARED_KEY=from_second
ONLY_IN_SECOND=value2
API_KEY=second_api_key
`
	err = os.WriteFile(file2, []byte(file2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create second file: %v", err)
	}

	file3 := filepath.Join(tmpDir, "third.env")
	file3Content := `# Third source
SHARED_KEY=from_third
ONLY_IN_THIRD=value3
API_KEY=third_api_key
`
	err = os.WriteFile(file3, []byte(file3Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create third file: %v", err)
	}

	// Create config with sources in specific order
	cfg := config.NewConfig()
	url1, _ := url.Parse("file://" + file1)
	url2, _ := url.Parse("file://" + file2)
	url3, _ := url.Parse("file://" + file3)

	cfg.Sources = append(cfg.Sources, &config.Source{URL: url1})
	cfg.Sources = append(cfg.Sources, &config.Source{URL: url2})
	cfg.Sources = append(cfg.Sources, &config.Source{URL: url3})

	// Call the REAL production function
	results := ResolveConfigSources(cfg)

	// Verify we got results in the same order as sources
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Verify each result corresponds to the correct source
	if results[0].Source.URL.String() != url1.String() {
		t.Errorf("Result[0] source mismatch: got %s, want %s", results[0].Source.URL.String(), url1.String())
	}
	if results[1].Source.URL.String() != url2.String() {
		t.Errorf("Result[1] source mismatch: got %s, want %s", results[1].Source.URL.String(), url2.String())
	}
	if results[2].Source.URL.String() != url3.String() {
		t.Errorf("Result[2] source mismatch: got %s, want %s", results[2].Source.URL.String(), url3.String())
	}

	// Now merge using the actual order returned by ResolveConfigSources
	var itemMaps []map[string]string
	for _, result := range results {
		if result.HasErrors() {
			t.Errorf("Result had errors: %v", result.Errors)
		}
		itemMaps = append(itemMaps, result.Items)
	}

	merged := formatters.Merge(itemMaps)

	// Verify the merge behavior - later sources should win
	expected := map[string]string{
		"ONLY_IN_FIRST":  "value1",
		"ONLY_IN_SECOND": "value2",
		"ONLY_IN_THIRD":  "value3",
		"DATABASE_HOST":  "localhost",
		"SHARED_KEY":     "from_third",       // Third source wins
		"API_KEY":        "third_api_key",    // Third source wins over second
	}

	if len(merged) != len(expected) {
		t.Errorf("Expected %d merged items, got %d", len(expected), len(merged))
		t.Logf("Expected: %+v", expected)
		t.Logf("Got: %+v", merged)
	}

	for key, expectedValue := range expected {
		if actualValue, ok := merged[key]; !ok {
			t.Errorf("Expected key %s not found in merged results", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Critical assertion: SHARED_KEY must be from the third source
	// This proves that order is preserved through the entire pipeline
	if merged["SHARED_KEY"] != "from_third" {
		t.Errorf("CRITICAL: Order not preserved! SHARED_KEY should be 'from_third', got %q", merged["SHARED_KEY"])
	}
}

// TestMultiSourceTypesMergeLogic demonstrates the merge logic with different source types.
// NOTE: This test uses mocks for sm:// and s3:// to show the concept, but does NOT test
// the actual ResolveConfigSources function with those source types (which would require AWS).
// See TestMultiSourceOrderPreservation for testing the real production code path.
func TestMultiSourceTypesMergeLogic(t *testing.T) {
	// This test demonstrates the CONCEPT of merging file://, sm://, and s3:// sources,
	// but it's NOT testing the actual production flow through ResolveConfigSources.
	// It's kept here for documentation purposes to show how different source types
	// would interact in a real scenario.

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

	// 1. Resolve file:// source (real resolver)
	fileURL, _ := url.Parse("file://" + envFile)
	fileSource := &config.Source{URL: fileURL}
	fileResolver := &resolvers.FileResolver{}
	fileResult := fileResolver.Resolve(fileSource)

	if fileResult.HasErrors() {
		t.Fatalf("File resolver failed: %v", fileResult.Errors)
	}

	// 2. Simulate sm:// source (mock - can't test without AWS)
	smURL, _ := url.Parse("sm://production/database/*")
	smSource := &config.Source{URL: smURL}
	mockSMResolver := &mockResolver{
		items: map[string]string{
			"DATABASE_PASSWORD": "secret_from_sm",
			"DATABASE_USER":     "admin",
			"SHARED_SECRET":     "from_sm", // Would overwrite file://
		},
	}
	smResult := mockSMResolver.Resolve(smSource)

	// 3. Simulate s3:// source (mock - can't test without AWS)
	s3URL, _ := url.Parse("s3://my-bucket/config.json")
	s3Source := &config.Source{URL: s3URL}
	mockS3Resolver := &mockResolver{
		items: map[string]string{
			"CACHE_ENDPOINT": "redis.example.com",
			"CACHE_PORT":     "6379",
			"API_KEY":        "s3_api_key",      // Would overwrite file://
			"SHARED_SECRET":  "from_s3_final",  // Would overwrite both file:// and sm://
		},
	}
	s3Result := mockS3Resolver.Resolve(s3Source)

	// Merge results (demonstrating the merge logic only)
	results := []*resolvers.Result{fileResult, smResult, s3Result}
	var itemMaps []map[string]string
	for _, result := range results {
		itemMaps = append(itemMaps, result.Items)
	}

	merged := formatters.Merge(itemMaps)

	// Verify the merge logic works as expected
	expected := map[string]string{
		"DATABASE_HOST":     "localhost",
		"DATABASE_PORT":     "5432",
		"DATABASE_PASSWORD": "secret_from_sm",
		"DATABASE_USER":     "admin",
		"CACHE_ENDPOINT":    "redis.example.com",
		"CACHE_PORT":        "6379",
		"API_KEY":           "s3_api_key",
		"SHARED_SECRET":     "from_s3_final",
	}

	for key, expectedValue := range expected {
		if actualValue, ok := merged[key]; !ok {
			t.Errorf("Expected key %s not found in merged results", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}
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
