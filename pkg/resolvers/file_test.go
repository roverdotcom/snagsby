package resolvers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "test.env")

	content := `# Comment line
KEY1=value1
KEY2="value2"
KEY3='value3'
KEY4=sm://production/secret

# Another comment
EMPTY_LINE_ABOVE=test
KEY_WITH_SPACES=value with spaces
`

	err := os.WriteFile(envFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to parse env file: %v", err)
	}

	// Expected items
	expected := map[string]struct {
		value           string
		needsResolution bool
	}{
		"KEY1":              {"value1", false},
		"KEY2":              {"value2", false},
		"KEY3":              {"value3", false},
		"KEY4":              {"sm://production/secret", true},
		"EMPTY_LINE_ABOVE":  {"test", false},
		"KEY_WITH_SPACES":   {"value with spaces", false},
	}

	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}

	for _, item := range items {
		exp, ok := expected[item.Key]
		if !ok {
			t.Errorf("Unexpected key: %s", item.Key)
			continue
		}

		if item.Value != exp.value {
			t.Errorf("Key %s: expected value %q, got %q", item.Key, exp.value, item.Value)
		}

		if item.NeedsResolution != exp.needsResolution {
			t.Errorf("Key %s: expected NeedsResolution=%v, got %v", item.Key, exp.needsResolution, item.NeedsResolution)
		}
	}
}

func TestParseEnvFileWithMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "malformed.env")

	content := `KEY1=value1
MALFORMED_NO_EQUALS
KEY2=value2
=no_key
KEY3=value3
`

	err := os.WriteFile(envFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to parse env file: %v", err)
	}

	// Should only parse valid lines
	if len(items) != 3 {
		t.Errorf("Expected 3 valid items, got %d", len(items))
	}

	validKeys := map[string]bool{"KEY1": true, "KEY2": true, "KEY3": true}
	for _, item := range items {
		if !validKeys[item.Key] {
			t.Errorf("Unexpected key parsed: %s", item.Key)
		}
	}
}

func TestParseEnvFileNotFound(t *testing.T) {
	_, err := parseEnvFile("/nonexistent/file.env")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestParseEnvFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "empty.env")

	err := os.WriteFile(envFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to parse env file: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items for empty file, got %d", len(items))
	}
}

func TestParseEnvFileOnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "comments.env")

	content := `# Comment 1
# Comment 2
# Comment 3
`

	err := os.WriteFile(envFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to parse env file: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items for file with only comments, got %d", len(items))
	}
}

func TestParseEnvFileEqualsInValue(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "equals.env")

	content := `KEY1=value=with=equals
KEY2="value=with=equals=in=quotes"
`

	err := os.WriteFile(envFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to parse env file: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	expected := map[string]string{
		"KEY1": "value=with=equals",
		"KEY2": "value=with=equals=in=quotes",
	}

	for _, item := range items {
		if exp, ok := expected[item.Key]; ok {
			if item.Value != exp {
				t.Errorf("Key %s: expected value %q, got %q", item.Key, exp, item.Value)
			}
		}
	}
}
