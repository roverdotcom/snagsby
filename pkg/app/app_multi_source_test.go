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
		"SHARED_KEY":     "from_third",    // Third source wins
		"API_KEY":        "third_api_key", // Third source wins over second
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

// TestMultiSourceTypesMergeLogic demonstrates how formatters.Merge works
// when combining results from different source types (file://, sm://, s3://).
func TestMultiSourceTypesMergeLogic(t *testing.T) {
	// Simulate results from three different source types
	fileURL, _ := url.Parse("file:///config/base.env")
	smURL, _ := url.Parse("sm://production/database/*")
	s3URL, _ := url.Parse("s3://my-bucket/config.json")

	// Result 1: from file:// (would contain static config)
	fileResult := &resolvers.Result{
		Source: &config.Source{URL: fileURL},
		Items: map[string]string{
			"DATABASE_HOST": "localhost",
			"DATABASE_PORT": "5432",
			"API_KEY":       "file_api_key",
			"SHARED_SECRET": "from_file",
		},
	}

	// Result 2: from sm:// (would contain AWS secrets)
	smResult := &resolvers.Result{
		Source: &config.Source{URL: smURL},
		Items: map[string]string{
			"DATABASE_PASSWORD": "secret_from_sm",
			"DATABASE_USER":     "admin",
			"SHARED_SECRET":     "from_sm", // Overwrites file://
		},
	}

	// Result 3: from s3:// (would contain final overrides)
	s3Result := &resolvers.Result{
		Source: &config.Source{URL: s3URL},
		Items: map[string]string{
			"CACHE_ENDPOINT": "redis.example.com",
			"CACHE_PORT":     "6379",
			"API_KEY":        "s3_api_key",    // Overwrites file://
			"SHARED_SECRET":  "from_s3_final", // Overwrites both file:// and sm://
		},
	}

	// Merge in order: file < sm < s3
	merged := formatters.Merge([]map[string]string{
		fileResult.Items,
		smResult.Items,
		s3Result.Items,
	})

	// Verify the merge behavior
	expected := map[string]string{
		"DATABASE_HOST":     "localhost",         // From file:// (not overwritten)
		"DATABASE_PORT":     "5432",              // From file:// (not overwritten)
		"DATABASE_PASSWORD": "secret_from_sm",    // From sm://
		"DATABASE_USER":     "admin",             // From sm://
		"CACHE_ENDPOINT":    "redis.example.com", // From s3://
		"CACHE_PORT":        "6379",              // From s3://
		"API_KEY":           "s3_api_key",        // From s3:// (overwrote file://)
		"SHARED_SECRET":     "from_s3_final",     // From s3:// (overwrote sm:// and file://)
	}

	for key, expectedValue := range expected {
		if actualValue, ok := merged[key]; !ok {
			t.Errorf("Expected key %s not found in merged results", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}

// TestRealWorldScenario simulates a typical layered configuration pattern:
// defaults < local overrides < production secrets
func TestRealWorldScenario(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env.defaults (committed to repo)
	defaultsFile := filepath.Join(tmpDir, ".env.defaults")
	defaultsContent := `APP_NAME=myapp
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
	localContent := `LOG_LEVEL=debug
DATABASE_HOST=127.0.0.1
`
	err = os.WriteFile(localFile, []byte(localContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}

	// Resolve defaults and local using real file resolver
	defaultsURL, _ := url.Parse("file://" + defaultsFile)
	defaultsResolver := &resolvers.FileResolver{}
	defaultsResult := defaultsResolver.Resolve(&config.Source{URL: defaultsURL})

	localURL, _ := url.Parse("file://" + localFile)
	localResolver := &resolvers.FileResolver{}
	localResult := localResolver.Resolve(&config.Source{URL: localURL})

	// Simulate AWS Secrets Manager result (would come from sm://production/api/*)
	smURL, _ := url.Parse("sm://production/api/*")
	smResult := &resolvers.Result{
		Source: &config.Source{URL: smURL},
		Items: map[string]string{
			"API_KEY":    "prod_api_key_from_aws",
			"API_SECRET": "prod_secret_from_aws",
		},
	}

	// Merge: defaults < local < aws (later wins)
	merged := formatters.Merge([]map[string]string{
		defaultsResult.Items,
		localResult.Items,
		smResult.Items,
	})

	// Verify layered configuration
	tests := []struct {
		key      string
		expected string
		source   string
	}{
		{"APP_NAME", "myapp", "defaults"},
		{"APP_ENV", "development", "defaults"},
		{"LOG_LEVEL", "debug", "local override"},
		{"DATABASE_HOST", "127.0.0.1", "local override"},
		{"DATABASE_PORT", "5432", "defaults"},
		{"CACHE_TTL", "300", "defaults"},
		{"API_KEY", "prod_api_key_from_aws", "AWS SM"},
		{"API_SECRET", "prod_secret_from_aws", "AWS SM"},
	}

	for _, tt := range tests {
		if actual, ok := merged[tt.key]; !ok {
			t.Errorf("Key %s not found (expected from %s)", tt.key, tt.source)
		} else if actual != tt.expected {
			t.Errorf("Key %s: expected %q (from %s), got %q", tt.key, tt.expected, tt.source, actual)
		}
	}
}

// TestSourceOrderMatters verifies that source order determines precedence
func TestSourceOrderMatters(t *testing.T) {
	tmpDir := t.TempDir()

	files := []struct {
		name  string
		value string
	}{
		{"first.env", "KEY=first\n"},
		{"second.env", "KEY=second\n"},
		{"third.env", "KEY=third\n"},
	}

	cfg := config.NewConfig()
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		os.WriteFile(path, []byte(f.value), 0644)
		url, _ := url.Parse("file://" + path)
		cfg.Sources = append(cfg.Sources, &config.Source{URL: url})
	}

	// Resolve using production code
	results := ResolveConfigSources(cfg)

	var itemMaps []map[string]string
	for _, result := range results {
		itemMaps = append(itemMaps, result.Items)
	}

	merged := formatters.Merge(itemMaps)

	// Last source should win
	if merged["KEY"] != "third" {
		t.Errorf("Expected last source to win: got %q, wanted %q", merged["KEY"], "third")
	}
}
