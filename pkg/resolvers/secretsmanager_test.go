package resolvers

import (
	"errors"
	"net/url"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
)

// mockSecretsManagerConnector implements SecretsManagerConnector for testing
type mockSecretsManagerConnector struct {
	listSecretsFunc func(prefix string) ([]string, error)
	getSecretFunc   func(secretName string) (string, error)
	getSecretsFunc  func(keys []string) (map[string]string, []error)
}

func (m *mockSecretsManagerConnector) ListSecrets(prefix string) ([]string, error) {
	if m.listSecretsFunc != nil {
		return m.listSecretsFunc(prefix)
	}
	return nil, nil
}

func (m *mockSecretsManagerConnector) GetSecret(secretName string) (string, error) {
	if m.getSecretFunc != nil {
		return m.getSecretFunc(secretName)
	}
	return "", nil
}

func (m *mockSecretsManagerConnector) GetSecrets(keys []string) (map[string]string, []error) {
	if m.getSecretsFunc != nil {
		return m.getSecretsFunc(keys)
	}
	return nil, nil
}

func TestIsRecursive(t *testing.T) {
	sm := &SecretsManagerResolver{}
	var url *url.URL
	var res bool
	var err error

	url, err = url.Parse("sm://hello/world/*")
	res = sm.isRecursive(&config.Source{URL: url})
	if res != true || err != nil {
		t.Errorf("Is recursive failed test %s", err)
	}

	url, err = url.Parse("sm://hello/world")
	res = sm.isRecursive(&config.Source{URL: url})
	if res == true || err != nil {
		t.Errorf("Is recursive failed test %s", err)
	}
}

func TestKeyNameFromPrefix(t *testing.T) {
	var val string
	sm := &SecretsManagerResolver{}
	val = sm.keyNameFromPrefix("/hello/", "/hello/charles-dickens")
	if val != "CHARLES_DICKENS" {
		t.Errorf("Should match CHARLES_DICKENS, but got %s", val)
	}
}

func TestResolveSingle(t *testing.T) {
	tests := []struct {
		name           string
		sourceURL      string
		mockGetSecret  func(secretName string) (string, error)
		expectError    bool
		expectedItems  map[string]string
		expectedErrMsg string
	}{
		{
			name:      "successful single secret retrieval",
			sourceURL: "sm://my-secret",
			mockGetSecret: func(secretName string) (string, error) {
				if secretName != "my-secret" {
					t.Errorf("Expected secretName 'my-secret', got '%s'", secretName)
				}
				return `{"DB_HOST":"localhost","DB_PORT":"5432"}`, nil
			},
			expectError: false,
			expectedItems: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
		},
		{
			name:      "successful single secret with path",
			sourceURL: "sm://prod/database/config",
			mockGetSecret: func(secretName string) (string, error) {
				if secretName != "prod/database/config" {
					t.Errorf("Expected secretName 'prod/database/config', got '%s'", secretName)
				}
				return `{"USERNAME":"admin","PASSWORD":"secret123"}`, nil
			},
			expectError: false,
			expectedItems: map[string]string{
				"USERNAME": "admin",
				"PASSWORD": "secret123",
			},
		},
		{
			name:      "GetSecret returns error",
			sourceURL: "sm://non-existent-secret",
			mockGetSecret: func(secretName string) (string, error) {
				return "", errors.New("secret not found")
			},
			expectError:    true,
			expectedErrMsg: "secret not found",
		},
		{
			name:      "invalid JSON in secret",
			sourceURL: "sm://invalid-json-secret",
			mockGetSecret: func(secretName string) (string, error) {
				return `{invalid json}`, nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConnector := &mockSecretsManagerConnector{
				getSecretFunc: tt.mockGetSecret,
			}

			resolver := &SecretsManagerResolver{
				connector: mockConnector,
			}

			parsedURL, err := url.Parse(tt.sourceURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			source := &config.Source{URL: parsedURL}
			result := resolver.resolveSingle(source)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error but got none")
				}
				if tt.expectedErrMsg != "" && result.Errors[0].Error() != tt.expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.expectedErrMsg, result.Errors[0].Error())
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected error: %v", result.Errors[0])
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
			}
		})
	}
}

func TestResolveRecursive(t *testing.T) {
	tests := []struct {
		name           string
		sourceURL      string
		mockListFunc   func(prefix string) ([]string, error)
		mockGetSecrets func(keys []string) (map[string]string, []error)
		expectError    bool
		expectedItems  map[string]string
	}{
		{
			name:      "successful recursive retrieval",
			sourceURL: "sm://prod/api/*",
			mockListFunc: func(prefix string) ([]string, error) {
				if prefix != "prod/api/" {
					t.Errorf("Expected prefix 'prod/api/', got '%s'", prefix)
				}
				return []string{
					"prod/api/key1",
					"prod/api/key2",
				}, nil
			},
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{
					"prod/api/key1": `{"value":"secret1"}`,
					"prod/api/key2": `{"value":"secret2"}`,
				}, nil
			},
			expectError: false,
			expectedItems: map[string]string{
				"KEY1": `{"value":"secret1"}`,
				"KEY2": `{"value":"secret2"}`,
			},
		},
		{
			name:      "ListSecrets returns error",
			sourceURL: "sm://prod/api/*",
			mockListFunc: func(prefix string) ([]string, error) {
				return nil, errors.New("failed to list secrets")
			},
			expectError: true,
		},
		{
			name:      "GetSecrets returns errors",
			sourceURL: "sm://prod/api/*",
			mockListFunc: func(prefix string) ([]string, error) {
				return []string{
					"prod/api/key1",
					"prod/api/key2",
				}, nil
			},
			mockGetSecrets: func(keys []string) (map[string]string, []error) {
				return map[string]string{
						"prod/api/key1": `{"value":"secret1"}`,
					}, []error{
						errors.New("failed to get prod/api/key2"),
					}
			},
			expectError: true,
			expectedItems: map[string]string{
				"KEY1": `{"value":"secret1"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConnector := &mockSecretsManagerConnector{
				listSecretsFunc: tt.mockListFunc,
				getSecretsFunc:  tt.mockGetSecrets,
			}

			resolver := &SecretsManagerResolver{
				connector: mockConnector,
			}

			parsedURL, err := url.Parse(tt.sourceURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			source := &config.Source{URL: parsedURL}
			result := resolver.resolveRecursive(source)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected error but got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected error: %v", result.Errors[0])
				}
			}

			if tt.expectedItems != nil {
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
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		sourceURL   string
		isRecursive bool
	}{
		{
			name:        "resolve calls resolveSingle for non-recursive URL",
			sourceURL:   "sm://my-secret",
			isRecursive: false,
		},
		{
			name:        "resolve calls resolveRecursive for recursive URL",
			sourceURL:   "sm://prod/api/*",
			isRecursive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConnector := &mockSecretsManagerConnector{
				getSecretFunc: func(secretName string) (string, error) {
					return `{"key":"value"}`, nil
				},
				listSecretsFunc: func(prefix string) ([]string, error) {
					return []string{"prod/api/key1"}, nil
				},
				getSecretsFunc: func(keys []string) (map[string]string, []error) {
					return map[string]string{"prod/api/key1": `{"value":"secret1"}`}, nil
				},
			}

			resolver := &SecretsManagerResolver{
				connector: mockConnector,
			}

			parsedURL, err := url.Parse(tt.sourceURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			source := &config.Source{URL: parsedURL}
			result := resolver.Resolve(source)

			if result == nil {
				t.Error("Expected result, got nil")
			}

			if len(result.Errors) > 0 {
				t.Errorf("Unexpected error: %v", result.Errors[0])
			}

			if len(result.Items) == 0 {
				t.Error("Expected items in result, got none")
			}
		})
	}
}
