package connectors

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/roverdotcom/snagsby/pkg/config"
)

// mockSecretsManagerClient is a mock implementation of the Secrets Manager client
type mockSecretsManagerClient struct {
	getSecretValueFunc func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	listSecretsFunc    func(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
}

type mockAWSSecretsManagerBehavior func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)

func GetMockSecretsManagerConnectorWithMocks(mock *mockSecretsManagerClient) *SecretsManagerConnector {
	sourceURL, _ := url.Parse("sm://test")
	source := &config.Source{URL: sourceURL}

	return &SecretsManagerConnector{source: source, secretsmanagerClient: mock}
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.getSecretValueFunc != nil {
		return m.getSecretValueFunc(ctx, params, optFns...)
	}
	return nil, errors.New("mock not implemented")
}

func (m *mockSecretsManagerClient) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	if m.listSecretsFunc != nil {
		return m.listSecretsFunc(ctx, params, optFns...)
	}
	return nil, errors.New("mock not implemented")
}

func makeMockSecretsManagerClient(secret string, err error) *mockSecretsManagerClient {
	return &mockSecretsManagerClient{
		getSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			if err != nil {
				return nil, err
			}
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(secret),
			}, nil
		},
		listSecretsFunc: func(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
			return &secretsmanager.ListSecretsOutput{
				SecretList: []types.SecretListEntry{
					{Name: aws.String("secret1")},
					{Name: aws.String("secret2")},
				},
			}, nil
		},
	}
}

// makeBehavior creates a mockAWSSecretsManagerBehavior from a simple function
// that takes a secretId and returns (secretValue, error)
func makeBehavior(fn func(secretId string) (string, error)) mockAWSSecretsManagerBehavior {
	return func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		secretValue, err := fn(*params.SecretId)
		if err != nil {
			return nil, err
		}
		return &secretsmanager.GetSecretValueOutput{
			SecretString: aws.String(secretValue),
		}, nil
	}
}

// successBehavior creates a mockAWSSecretsManagerBehavior for success cases only
// where you just need to return a value based on the secretId
func successBehavior(fn func(secretId string) string) mockAWSSecretsManagerBehavior {
	return makeBehavior(func(secretId string) (string, error) {
		return fn(secretId), nil
	})
}

// errorBehavior creates a mockAWSSecretsManagerBehavior that always returns an error
func errorBehavior(err error) mockAWSSecretsManagerBehavior {
	return makeBehavior(func(secretId string) (string, error) {
		return "", err
	})
}

// TestGetSecretWithVersionStage tests that version-stage query parameter is passed correctly
func TestGetSecretWithVersionStage(t *testing.T) {
	versionStageReceived := ""
	mockClient := &mockSecretsManagerClient{
		getSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			if params.VersionStage != nil {
				versionStageReceived = *params.VersionStage
			}
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("test-value"),
			}, nil
		},
	}

	sourceURL, _ := url.Parse("sm://test?version-stage=AWSCURRENT")
	source := &config.Source{URL: sourceURL}
	sm := &SecretsManagerConnector{source: source, secretsmanagerClient: mockClient}

	result, err := sm.GetSecret("test-secret")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got %s", result)
	}
	if versionStageReceived != "AWSCURRENT" {
		t.Errorf("Expected version-stage AWSCURRENT, got %s", versionStageReceived)
	}
}

// TestGetSecretWithVersionID tests that version-id query parameter is passed correctly
func TestGetSecretWithVersionID(t *testing.T) {
	versionIDReceived := ""
	mockClient := &mockSecretsManagerClient{
		getSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			if params.VersionId != nil {
				versionIDReceived = *params.VersionId
			}
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("test-value"),
			}, nil
		},
	}

	sourceURL, _ := url.Parse("sm://test?version-id=abc123")
	source := &config.Source{URL: sourceURL}
	sm := &SecretsManagerConnector{source: source, secretsmanagerClient: mockClient}

	result, err := sm.GetSecret("test-secret")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got %s", result)
	}
	if versionIDReceived != "abc123" {
		t.Errorf("Expected version-id abc123, got %s", versionIDReceived)
	}
}

// TestGetSecretErrors tests error handling in GetSecret
func TestGetSecretErrors(t *testing.T) {
	tests := []struct {
		name          string
		mockBehavior  mockAWSSecretsManagerBehavior
		expectedError string
	}{
		{
			name:          "access denied error",
			mockBehavior:  errorBehavior(errors.New("access denied")),
			expectedError: "access denied",
		},
		{
			name:          "secret not found error",
			mockBehavior:  errorBehavior(errors.New("secret not found")),
			expectedError: "secret not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockSecretsManagerClient{
				getSecretValueFunc: tt.mockBehavior,
			}
			sm := GetMockSecretsManagerConnectorWithMocks(mockClient)

			_, err := sm.GetSecret("test-secret")

			if err == nil {
				t.Error("Expected error but got none")
			} else if err.Error() != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

// TestGetSecrets tests the getSecrets function
func TestGetSecrets(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		mockBehavior  mockAWSSecretsManagerBehavior
		expectedItems int
		expectedError bool
	}{
		{
			name:          "single secret success",
			keys:          []string{"secret1"},
			mockBehavior:  successBehavior(func(secretId string) string { return "value1" }),
			expectedItems: 1,
			expectedError: false,
		},
		{
			name:          "multiple secrets success",
			keys:          []string{"secret1", "secret2", "secret3"},
			mockBehavior:  successBehavior(func(secretId string) string { return "value-" + secretId }),
			expectedItems: 3,
			expectedError: false,
		},
		{
			name: "partial failure",
			keys: []string{"secret1", "secret2", "secret3"},
			mockBehavior: makeBehavior(func(secretId string) (string, error) {
				if secretId == "secret2" {
					return "", errors.New("secret not found")
				}
				return "value-" + secretId, nil
			}),
			expectedItems: 2,
			expectedError: true,
		},
		{
			name:          "complete failure - all secrets fail",
			keys:          []string{"secret1", "secret2", "secret3"},
			mockBehavior:  errorBehavior(errors.New("access denied")),
			expectedItems: 0,
			expectedError: true,
		},
		{
			name:          "empty keys",
			keys:          []string{},
			mockBehavior:  successBehavior(func(secretId string) string { return "should-not-be-called" }),
			expectedItems: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockSecretsManagerClient{
				getSecretValueFunc: tt.mockBehavior,
			}
			sm := GetMockSecretsManagerConnectorWithMocks(mockClient)
			result, errors := sm.GetSecrets(tt.keys)

			if len(result) != tt.expectedItems {
				t.Errorf("Expected %d items, got %d", tt.expectedItems, len(result))
			}

			if tt.expectedError && len(errors) == 0 {
				t.Error("Expected errors but got none")
			}

			if !tt.expectedError && len(errors) > 0 {
				t.Errorf("Unexpected errors: %v", errors)
			}
		})
	}
}

// TestGetConcurrencyOrDefault tests the GetConcurrencyOrDefault function
func TestGetConcurrencyOrDefault(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		keyLength int
		expected  int
	}{
		{
			name:      "no env var set - returns keyLength",
			envValue:  "",
			keyLength: 10,
			expected:  10,
		},
		{
			name:      "valid env var set",
			envValue:  "5",
			keyLength: 10,
			expected:  5,
		},
		{
			name:      "env var set to 0",
			envValue:  "0",
			keyLength: 10,
			expected:  10,
		},
		{
			name:      "invalid env var - returns keyLength",
			envValue:  "invalid",
			keyLength: 8,
			expected:  8,
		},
		{
			name:      "negative env var - returns keyLength",
			envValue:  "-1",
			keyLength: 7,
			expected:  7,
		},
		{
			name:      "env var larger than keyLength",
			envValue:  "20",
			keyLength: 5,
			expected:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smc := &SecretsManagerConnector{}

			// Save and restore environment variable
			originalEnv, hadEnv := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
			defer func() {
				if hadEnv {
					os.Setenv("SNAGSBY_SM_CONCURRENCY", originalEnv)
				} else {
					os.Unsetenv("SNAGSBY_SM_CONCURRENCY")
				}
			}()

			// Set test environment variable
			if tt.envValue != "" {
				os.Setenv("SNAGSBY_SM_CONCURRENCY", tt.envValue)
			} else {
				os.Unsetenv("SNAGSBY_SM_CONCURRENCY")
			}

			result := smc.getConcurrencyOrDefault(tt.keyLength)

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestGetSecretsConcurrency tests that concurrency settings are properly used in GetSecrets
func TestGetSecretsConcurrency(t *testing.T) {
	tests := []struct {
		name                string
		envValue            string
		numKeys             int
		expectedConcurrency int // What getConcurrencyOrDefault should return
		expectedMaxWorkers  int // Maximum number of concurrent workers actually processing
	}{
		{
			name:                "no concurrency limit - uses number of keys",
			envValue:            "",
			numKeys:             5,
			expectedConcurrency: 5,
			expectedMaxWorkers:  5,
		},
		{
			name:                "concurrency set to 2",
			envValue:            "2",
			numKeys:             10,
			expectedConcurrency: 2,
			expectedMaxWorkers:  2,
		},
		{
			name:                "concurrency set to 1 (sequential)",
			envValue:            "1",
			numKeys:             5,
			expectedConcurrency: 1,
			expectedMaxWorkers:  1,
		},
		{
			name:                "concurrency larger than keys",
			envValue:            "20",
			numKeys:             3,
			expectedConcurrency: 20,
			expectedMaxWorkers:  3, // Only 3 keys, so at most 3 workers will be active
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment variable
			originalEnv, hadEnv := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
			defer func() {
				if hadEnv {
					os.Setenv("SNAGSBY_SM_CONCURRENCY", originalEnv)
				} else {
					os.Unsetenv("SNAGSBY_SM_CONCURRENCY")
				}
			}()

			// Set test environment variable
			if tt.envValue != "" {
				os.Setenv("SNAGSBY_SM_CONCURRENCY", tt.envValue)
			} else {
				os.Unsetenv("SNAGSBY_SM_CONCURRENCY")
			}

			smc := &SecretsManagerConnector{}

			// Verify getConcurrencyOrDefault returns expected value
			actualConcurrency := smc.getConcurrencyOrDefault(tt.numKeys)
			if actualConcurrency != tt.expectedConcurrency {
				t.Errorf("getConcurrencyOrDefault(%d) = %d, expected %d", tt.numKeys, actualConcurrency, tt.expectedConcurrency)
			}

			// Create keys
			keys := make([]string, tt.numKeys)
			for i := 0; i < tt.numKeys; i++ {
				keys[i] = "secret" + strconv.Itoa(i)
			}

			// Track concurrent calls
			var concurrentCalls int32
			var maxConcurrent int32

			mockClient := &mockSecretsManagerClient{
				getSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					// Atomically increment concurrent calls
					current := atomic.AddInt32(&concurrentCalls, 1)

					// Track max concurrent
					for {
						max := atomic.LoadInt32(&maxConcurrent)
						if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
							break
						}
					}

					// Simulate some work
					time.Sleep(10 * time.Millisecond)

					// Decrement
					atomic.AddInt32(&concurrentCalls, -1)

					return &secretsmanager.GetSecretValueOutput{
						SecretString: aws.String("value"),
					}, nil
				},
			}

			sm := GetMockSecretsManagerConnectorWithMocks(mockClient)
			result, errors := sm.GetSecrets(keys)

			// Verify all secrets were retrieved
			if len(result) != tt.numKeys {
				t.Errorf("Expected %d items, got %d", tt.numKeys, len(result))
			}

			if len(errors) > 0 {
				t.Errorf("Unexpected errors: %v", errors)
			}

			// Verify concurrency was respected
			finalMax := atomic.LoadInt32(&maxConcurrent)
			if tt.expectedMaxWorkers > 0 && finalMax > int32(tt.expectedMaxWorkers) {
				t.Errorf("Expected max %d concurrent workers, but saw %d", tt.expectedMaxWorkers, finalMax)
			}
		})
	}
}
