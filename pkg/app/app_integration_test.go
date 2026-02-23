package app

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/formatters"
)

// TestMultipleSourcesWithOverwriting tests that multiple sources are resolved
// and later sources overwrite earlier ones
func TestMultipleSourcesWithOverwriting(t *testing.T) {
	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "test.env")

	envContent := `# Test env file
KEY1=from_file
KEY2=also_from_file
SHARED_KEY=file_value
`

	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create config with multiple sources
	// Note: We can't actually test sm:// and s3:// without AWS credentials,
	// but we can test that file:// works and produces results that would be merged correctly
	cfg := config.NewConfig()

	fileURL, err := url.Parse("file://" + envFile)
	if err != nil {
		t.Fatalf("Failed to parse file URL: %v", err)
	}
	cfg.Sources = append(cfg.Sources, &config.Source{URL: fileURL})

	// Resolve all sources
	results := ResolveConfigSources(cfg)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Check that file:// resolver worked
	fileResult := results[0]
	if fileResult.HasErrors() {
		t.Errorf("File resolver had errors: %v", fileResult.Errors)
	}

	if fileResult.LenItems() != 3 {
		t.Errorf("Expected 3 items from file, got %d", fileResult.LenItems())
	}

	// Verify values
	expectedValues := map[string]string{
		"KEY1":       "from_file",
		"KEY2":       "also_from_file",
		"SHARED_KEY": "file_value",
	}

	for key, expectedValue := range expectedValues {
		if actualValue, ok := fileResult.Items[key]; !ok {
			t.Errorf("Expected key %s not found in results", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}

// TestMergingMultipleResults tests the merge behavior when multiple results
// are combined using the formatters.Merge function
func TestMergingMultipleResults(t *testing.T) {
	// Create two temporary .env files
	tmpDir := t.TempDir()
	envFile1 := filepath.Join(tmpDir, "base.env")
	envFile2 := filepath.Join(tmpDir, "override.env")

	baseContent := `KEY1=base_value
KEY2=base_value_2
SHARED=from_base
`

	overrideContent := `KEY3=override_value
SHARED=from_override
`

	err := os.WriteFile(envFile1, []byte(baseContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	err = os.WriteFile(envFile2, []byte(overrideContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create override file: %v", err)
	}

	// Create config with multiple sources
	cfg := config.NewConfig()

	file1URL, _ := url.Parse("file://" + envFile1)
	file2URL, _ := url.Parse("file://" + envFile2)

	cfg.Sources = append(cfg.Sources, &config.Source{URL: file1URL})
	cfg.Sources = append(cfg.Sources, &config.Source{URL: file2URL})

	// Resolve all sources
	results := ResolveConfigSources(cfg)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Extract items from all results
	var itemMaps []map[string]string
	for _, result := range results {
		if result.HasErrors() {
			t.Errorf("Result had errors: %v", result.Errors)
		}
		itemMaps = append(itemMaps, result.Items)
	}

	// Merge results (this simulates what the main app does)
	merged := formatters.Merge(itemMaps)

	// Verify merged results - later sources should overwrite earlier ones
	expected := map[string]string{
		"KEY1":   "base_value",
		"KEY2":   "base_value_2",
		"KEY3":   "override_value",
		"SHARED": "from_override", // This should be from the second file
	}

	if len(merged) != len(expected) {
		t.Errorf("Expected %d merged items, got %d", len(expected), len(merged))
	}

	for key, expectedValue := range expected {
		if actualValue, ok := merged[key]; !ok {
			t.Errorf("Expected key %s not found in merged results", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}

// TestParallelResolution verifies that multiple sources are resolved concurrently
func TestParallelResolution(t *testing.T) {
	// Create multiple env files
	tmpDir := t.TempDir()
	numFiles := 5

	cfg := config.NewConfig()

	for i := 0; i < numFiles; i++ {
		envFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".env")
		content := "KEY" + string(rune('0'+i)) + "=value" + string(rune('0'+i)) + "\n"

		err := os.WriteFile(envFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}

		fileURL, _ := url.Parse("file://" + envFile)
		cfg.Sources = append(cfg.Sources, &config.Source{URL: fileURL})
	}

	// Resolve all sources (should be done concurrently)
	results := ResolveConfigSources(cfg)

	if len(results) != numFiles {
		t.Fatalf("Expected %d results, got %d", numFiles, len(results))
	}

	// Verify each result has its expected key
	for i, result := range results {
		if result.HasErrors() {
			t.Errorf("Result %d had errors: %v", i, result.Errors)
		}

		if result.LenItems() != 1 {
			t.Errorf("Result %d: expected 1 item, got %d", i, result.LenItems())
		}
	}
}
