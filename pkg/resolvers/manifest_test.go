package resolvers

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
)

// mockManifestConnector implements manifestSecretsConnector for testing
type mockManifestConnector struct {
	getSecretsFunc func(keys []string) (map[string]string, []error)
}

func (m *mockManifestConnector) GetSecrets(keys []string) (map[string]string, []error) {
	if m.getSecretsFunc != nil {
		return m.getSecretsFunc(keys)
	}
	return nil, nil
}

func TestManifestResolve(t *testing.T) {
	tests := []struct {
		name           string
		manifestYAML   string
		mockGetSecrets func(keys []string) (map[string]string, []error)
		expectError    bool
		expectedItems  map[string]string
	}{
		{
			name: "successful manifest resolution",
			manifestYAML: `items:
  - name: prod/api/database
    env: DATABASE_URL
  - name: prod/api/secret-key
    env: SECRET_KEY
`,
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				// Verify the keys requested
				if len(keys) != 2 {
					t.Errorf("Expected 2 secret keys, got %d", len(keys))
				}
				return map[string]string{
					"prod/api/database":   "postgres://localhost:5432/db",
					"prod/api/secret-key": "my-secret-key-123",
				}, nil
			},
			expectError: false,
			expectedItems: map[string]string{
				"DATABASE_URL": "postgres://localhost:5432/db",
				"SECRET_KEY":   "my-secret-key-123",
			},
		},
		{
			name: "empty manifest",
			manifestYAML: `items: []
`,
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				if len(keys) != 0 {
					t.Errorf("Expected 0 secret keys for empty manifest, got %d", len(keys))
				}
				return map[string]string{}, nil
			},
			expectError:   false,
			expectedItems: map[string]string{},
		},
		{
			name: "connector returns errors",
			manifestYAML: `items:
  - name: prod/api/database
    env: DATABASE_URL
  - name: prod/api/secret-key
    env: SECRET_KEY
`,
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{
						"prod/api/database": "postgres://localhost:5432/db",
					}, []error{
						errors.New("failed to get prod/api/secret-key"),
					}
			},
			expectError: true,
			expectedItems: map[string]string{
				"DATABASE_URL": "postgres://localhost:5432/db",
			},
		},
		{
			name: "multiple secrets with same env var (last one wins)",
			manifestYAML: `items:
  - name: prod/api/database-primary
    env: DATABASE_URL
  - name: prod/api/database-replica
    env: DATABASE_URL
`,
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{
					"prod/api/database-primary": "postgres://primary:5432/db",
					"prod/api/database-replica": "postgres://replica:5432/db",
				}, nil
			},
			expectError: false,
			expectedItems: map[string]string{
				// Last item in manifest wins due to deterministic iteration order
				"DATABASE_URL": "postgres://replica:5432/db",
			},
		},
		{
			name: "all secrets fail",
			manifestYAML: `items:
  - name: prod/api/database
    env: DATABASE_URL
`,
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{}, []error{
					errors.New("failed to get prod/api/database"),
				}
			},
			expectError:   true,
			expectedItems: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary manifest file
			tmpDir := t.TempDir()
			manifestPath := filepath.Join(tmpDir, "manifest.yaml")
			err := os.WriteFile(manifestPath, []byte(tt.manifestYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to create test manifest file: %v", err)
			}

			mockConnector := &mockManifestConnector{
				getSecretsFunc: tt.mockGetSecrets,
			}

			resolver := &ManifestResolver{
				connector: mockConnector,
			}

			sourceURL, err := url.Parse("manifest://" + manifestPath)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			source := &config.Source{URL: sourceURL}
			result := resolver.Resolve(source)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error but got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected error: %v", result.Errors[0])
				}
			}

			if len(result.Items) != len(tt.expectedItems) {
				t.Errorf("Expected %d items, got %d", len(tt.expectedItems), len(result.Items))
			}

			for key, expectedValue := range tt.expectedItems {
				if value, ok := result.Items[key]; !ok {
					t.Errorf("Expected key '%s' not found in result", key)
				} else if value != expectedValue {
					t.Errorf("For key '%s', expected value '%s', got '%s'", key, expectedValue, value)
				}
			}
		})
	}
}

func TestManifestResolveFileErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "file does not exist",
			setupFile: func(t *testing.T) string {
				return "/nonexistent/path/manifest.yaml"
			},
			expectError: true,
		},
		{
			name: "invalid YAML",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				manifestPath := filepath.Join(tmpDir, "manifest.yaml")
				err := os.WriteFile(manifestPath, []byte("invalid: yaml: content: ["), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return manifestPath
			},
			expectError: true,
		},
		{
			name: "empty file",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				manifestPath := filepath.Join(tmpDir, "manifest.yaml")
				err := os.WriteFile(manifestPath, []byte(""), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return manifestPath
			},
			expectError: false, // Empty YAML is valid, just results in empty items
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifestPath := tt.setupFile(t)

			mockConnector := &mockManifestConnector{
				getSecretsFunc: func(keys []string) (map[string]string, []error) {
					return map[string]string{}, nil
				},
			}

			resolver := &ManifestResolver{
				connector: mockConnector,
			}

			sourceURL, err := url.Parse("manifest://" + manifestPath)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			source := &config.Source{URL: sourceURL}
			result := resolver.Resolve(source)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error but got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected error: %v", result.Errors[0])
				}
			}
		})
	}
}

func TestResolveManifestItems(t *testing.T) {
	tests := []struct {
		name           string
		manifestItems  *ManifestItems
		mockGetSecrets func(keys []string) (map[string]string, []error)
		expectError    bool
		expectedItems  map[string]string
	}{
		{
			name: "successful resolution with multiple items",
			manifestItems: &ManifestItems{
				Items: []*ManifestItem{
					{Name: "secret1", Env: "ENV_VAR_1"},
					{Name: "secret2", Env: "ENV_VAR_2"},
					{Name: "secret3", Env: "ENV_VAR_3"},
				},
			},
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				if len(keys) != 3 {
					t.Errorf("Expected 3 keys, got %d", len(keys))
				}
				return map[string]string{
					"secret1": "value1",
					"secret2": "value2",
					"secret3": "value3",
				}, nil
			},
			expectError: false,
			expectedItems: map[string]string{
				"ENV_VAR_1": "value1",
				"ENV_VAR_2": "value2",
				"ENV_VAR_3": "value3",
			},
		},
		{
			name: "partial success with some errors",
			manifestItems: &ManifestItems{
				Items: []*ManifestItem{
					{Name: "secret1", Env: "ENV_VAR_1"},
					{Name: "secret2", Env: "ENV_VAR_2"},
				},
			},
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{
						"secret1": "value1",
					}, []error{
						errors.New("failed to get secret2"),
					}
			},
			expectError: true,
			expectedItems: map[string]string{
				"ENV_VAR_1": "value1",
			},
		},
		{
			name: "no items",
			manifestItems: &ManifestItems{
				Items: []*ManifestItem{},
			},
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{}, nil
			},
			expectError:   false,
			expectedItems: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConnector := &mockManifestConnector{
				getSecretsFunc: tt.mockGetSecrets,
			}

			resolver := &ManifestResolver{
				connector: mockConnector,
			}

			result := &Result{Items: make(map[string]string)}
			resolver.resolveManifestItems(tt.manifestItems, result)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error but got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected error: %v", result.Errors[0])
				}
			}

			if len(result.Items) != len(tt.expectedItems) {
				t.Errorf("Expected %d items, got %d", len(tt.expectedItems), len(result.Items))
			}

			for key, expectedValue := range tt.expectedItems {
				if value, ok := result.Items[key]; !ok {
					t.Errorf("Expected key '%s' not found in result", key)
				} else if value != expectedValue {
					t.Errorf("For key '%s', expected value '%s', got '%s'", key, expectedValue, value)
				}
			}
		})
	}
}

func TestManifestIntegrationWithResolveSource(t *testing.T) {
	// Create a temporary manifest file
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.yaml")
	manifestYAML := `items:
  - name: prod/api/database
    env: DATABASE_URL
  - name: prod/api/secret-key
    env: SECRET_KEY
`
	err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test manifest file: %v", err)
	}

	// Use a mock connector to avoid real AWS calls while exercising manifest resolution
	mockConnector := &mockManifestConnector{
		getSecretsFunc: func(keys []string) (map[string]string, []error) {
			return map[string]string{
				"prod/api/database":   "postgres://user:pass@host:5432/db",
				"prod/api/secret-key": "super-secret-key",
			}, nil
		},
	}

	resolver := &ManifestResolver{
		connector: mockConnector,
	}

	sourceURL, err := url.Parse("manifest://" + manifestPath)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	source := &config.Source{URL: sourceURL}
	result := resolver.Resolve(source)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Errors) > 0 {
		t.Fatalf("Unexpected errors: %v", result.Errors)
	}

	if got := result.Items["DATABASE_URL"]; got != "postgres://user:pass@host:5432/db" {
		t.Errorf("DATABASE_URL mismatch, got %q", got)
	}

	if got := result.Items["SECRET_KEY"]; got != "super-secret-key" {
		t.Errorf("SECRET_KEY mismatch, got %q", got)
	}
}

func TestManifestWithSpecialCharacters(t *testing.T) {
	manifestYAML := `items:
  - name: prod/api/special-chars
    env: SPECIAL_KEY_WITH_DASHES
  - name: prod/api/underscores_test
    env: UNDERSCORE_KEY
`
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.yaml")
	err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test manifest file: %v", err)
	}

	mockConnector := &mockManifestConnector{
		getSecretsFunc: func(keys []string) (map[string]string, []error) {
			return map[string]string{
				"prod/api/special-chars":    "value-with-dashes",
				"prod/api/underscores_test": "value_with_underscores",
			}, nil
		},
	}

	resolver := &ManifestResolver{
		connector: mockConnector,
	}

	sourceURL, err := url.Parse("manifest://" + manifestPath)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	source := &config.Source{URL: sourceURL}
	result := resolver.Resolve(source)

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected error: %v", result.Errors[0])
	}

	// Keys should be normalized by AppendItem
	if result.Items["SPECIAL_KEY_WITH_DASHES"] != "value-with-dashes" {
		t.Errorf("Expected SPECIAL_KEY_WITH_DASHES, got %v", result.Items)
	}

	if result.Items["UNDERSCORE_KEY"] != "value_with_underscores" {
		t.Errorf("Expected UNDERSCORE_KEY, got %v", result.Items)
	}
}

func TestManifestKeysAreCorrectlyPassed(t *testing.T) {
	manifestYAML := `items:
  - name: first-secret
    env: FIRST
  - name: second-secret
    env: SECOND
  - name: third-secret
    env: THIRD
`
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.yaml")
	err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test manifest file: %v", err)
	}

	// Verify the exact keys that are passed to GetSecrets
	var receivedKeys []string
	mockConnector := &mockManifestConnector{
		getSecretsFunc: func(keys []string) (map[string]string, []error) {
			receivedKeys = keys
			return map[string]string{
				"first-secret":  "value1",
				"second-secret": "value2",
				"third-secret":  "value3",
			}, nil
		},
	}

	resolver := &ManifestResolver{
		connector: mockConnector,
	}

	sourceURL, err := url.Parse("manifest://" + manifestPath)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	source := &config.Source{URL: sourceURL}
	result := resolver.Resolve(source)

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected error: %v", result.Errors[0])
	}

	if len(receivedKeys) != 3 {
		t.Errorf("Expected 3 keys to be passed to GetSecrets, got %d", len(receivedKeys))
	}

	// Create a map to check all keys were passed
	keyMap := make(map[string]bool)
	for _, key := range receivedKeys {
		keyMap[key] = true
	}

	expectedKeys := []string{"first-secret", "second-secret", "third-secret"}
	for _, expectedKey := range expectedKeys {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key '%s' was not passed to GetSecrets", expectedKey)
		}
	}
}
