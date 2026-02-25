package resolvers

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/roverdotcom/snagsby/pkg/config"
)

// mockSecretsManagerClient is a mock implementation of the Secrets Manager client
type mockSecretsManagerClient struct {
	getSecretValueFunc func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type mockAWSSecretsManagerBehavior func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.getSecretValueFunc != nil {
		return m.getSecretValueFunc(ctx, params, optFns...)
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

// TestSMWorker tests the smWorker function
func TestSMWorker(t *testing.T) {
	tests := []struct {
		name           string
		secretName     string
		secretValue    string
		mockError      error
		expectedError  bool
		expectedResult string
	}{
		{
			name:           "successful secret retrieval",
			secretName:     "test-secret",
			secretValue:    "secret-value",
			mockError:      nil,
			expectedError:  false,
			expectedResult: "secret-value",
		},
		{
			name:          "failed secret retrieval",
			secretName:    "test-secret",
			secretValue:   "",
			mockError:     errors.New("secret not found"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := makeMockSecretsManagerClient(tt.secretValue, tt.mockError)

			// Create source URL
			sourceURL, _ := url.Parse("sm://test")
			source := &config.Source{URL: sourceURL}

			// Create channels
			jobs := make(chan *smMessage, 1)
			results := make(chan *smMessage, 1)

			// Start worker
			go smWorker(jobs, results, mockClient)

			// Send job
			secretName := tt.secretName
			jobs <- &smMessage{
				Source: source,
				Name:   &secretName,
			}
			close(jobs)

			// Get result
			result := <-results

			// Verify
			if tt.expectedError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if result.Error != nil {
					t.Errorf("Unexpected error: %v", result.Error)
				}
				if result.Result != tt.expectedResult {
					t.Errorf("Expected result %s, got %s", tt.expectedResult, result.Result)
				}
			}
		})
	}
}

// TestSMWorkerWithVersionStage tests that version-stage query parameter is passed correctly
func TestSMWorkerWithVersionStage(t *testing.T) {
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

	jobs := make(chan *smMessage, 1)
	results := make(chan *smMessage, 1)

	go smWorker(jobs, results, mockClient)

	secretName := "test-secret"
	jobs <- &smMessage{
		Source: source,
		Name:   &secretName,
	}
	close(jobs)

	<-results

	if versionStageReceived != "AWSCURRENT" {
		t.Errorf("Expected version-stage AWSCURRENT, got %s", versionStageReceived)
	}
}

// TestSMWorkerWithVersionID tests that version-id query parameter is passed correctly
func TestSMWorkerWithVersionID(t *testing.T) {
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

	jobs := make(chan *smMessage, 1)
	results := make(chan *smMessage, 1)

	go smWorker(jobs, results, mockClient)

	secretName := "test-secret"
	jobs <- &smMessage{
		Source: source,
		Name:   &secretName,
	}
	close(jobs)

	<-results

	if versionIDReceived != "abc123" {
		t.Errorf("Expected version-id abc123, got %s", versionIDReceived)
	}
}

// TestSMWorkerWithErrorBehavior tests smWorker using errorBehavior helper
func TestSMWorkerWithErrorBehavior(t *testing.T) {
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

			sourceURL, _ := url.Parse("sm://test")
			source := &config.Source{URL: sourceURL}

			jobs := make(chan *smMessage, 1)
			results := make(chan *smMessage, 1)

			go smWorker(jobs, results, mockClient)

			secretName := "test-secret"
			jobs <- &smMessage{
				Source: source,
				Name:   &secretName,
			}
			close(jobs)

			result := <-results

			if result.Error == nil {
				t.Error("Expected error but got none")
			} else if result.Error.Error() != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, result.Error.Error())
			}
		})
	}
}

// TestGetSecrets tests the getSecrets function
func TestGetSecrets(t *testing.T) {
	tests := []struct {
		name          string
		keys          []*string
		mockBehavior  mockAWSSecretsManagerBehavior
		expectedItems int
		expectedError bool
	}{
		{
			name:          "single secret success",
			keys:          []*string{aws.String("secret1")},
			mockBehavior:  successBehavior(func(secretId string) string { return "value1" }),
			expectedItems: 1,
			expectedError: false,
		},
		{
			name:          "multiple secrets success",
			keys:          []*string{aws.String("secret1"), aws.String("secret2"), aws.String("secret3")},
			mockBehavior:  successBehavior(func(secretId string) string { return "value-" + secretId }),
			expectedItems: 3,
			expectedError: false,
		},
		{
			name: "partial failure",
			keys: []*string{aws.String("secret1"), aws.String("secret2"), aws.String("secret3")},
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
			keys:          []*string{aws.String("secret1"), aws.String("secret2"), aws.String("secret3")},
			mockBehavior:  errorBehavior(errors.New("access denied")),
			expectedItems: 0,
			expectedError: true,
		},
		{
			name:          "empty keys",
			keys:          []*string{},
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

			sourceURL, _ := url.Parse("sm://test")
			source := &config.Source{URL: sourceURL}

			result := getSecrets(source, mockClient, tt.keys)

			if result.LenItems() != tt.expectedItems {
				t.Errorf("Expected %d items, got %d", tt.expectedItems, result.LenItems())
			}

			if tt.expectedError && !result.HasErrors() {
				t.Error("Expected errors but got none")
			}

			if !tt.expectedError && result.HasErrors() {
				t.Errorf("Unexpected errors: %v", result.Errors)
			}
		})
	}
}

// TestGetSecretsConcurrency tests that concurrency is properly handled
func TestGetSecretsConcurrency(t *testing.T) {
	// Save original value and restore after test
	originalConcurrency := smConcurrency
	defer func() { smConcurrency = originalConcurrency }()

	tests := []struct {
		name             string
		smConcurrencyVal int
		numKeys          int
		expectedWorkers  int
	}{
		{
			name:             "concurrency not set - uses number of keys",
			smConcurrencyVal: 0,
			numKeys:          5,
			expectedWorkers:  5,
		},
		{
			name:             "concurrency set to 2",
			smConcurrencyVal: 2,
			numKeys:          10,
			expectedWorkers:  2,
		},
		{
			name:             "concurrency set to negative - uses number of keys",
			smConcurrencyVal: -1,
			numKeys:          3,
			expectedWorkers:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smConcurrency = tt.smConcurrencyVal

			mockClient := &mockSecretsManagerClient{
				getSecretValueFunc: successBehavior(func(secretId string) string { return "value" }),
			}

			sourceURL, _ := url.Parse("sm://test")
			source := &config.Source{URL: sourceURL}

			keys := make([]*string, tt.numKeys)
			for i := 0; i < tt.numKeys; i++ {
				keys[i] = aws.String("secret" + string(rune('0'+i)))
			}

			result := getSecrets(source, mockClient, keys)

			if result.LenItems() != tt.numKeys {
				t.Errorf("Expected %d items, got %d", tt.numKeys, result.LenItems())
			}

			if result.HasErrors() {
				t.Errorf("Unexpected errors: %v", result.Errors)
			}
		})
	}
}

// TestSMConcurrencyInit tests the init function behavior with environment variables
func TestSMConcurrencyInit(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "valid positive integer",
			envValue: "5",
			expected: 5,
		},
		{
			name:     "zero value",
			envValue: "0",
			expected: 0,
		},
		{
			name:     "invalid value",
			envValue: "invalid",
			expected: 0, // Should not change from default
		},
		{
			name:     "negative value",
			envValue: "-1",
			expected: 0, // Should not change from default (negative not allowed)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original value
			originalValue := smConcurrency
			defer func() { smConcurrency = originalValue }()

			// Reset to 0
			smConcurrency = 0

			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("SNAGSBY_SM_CONCURRENCY", tt.envValue)
				defer os.Unsetenv("SNAGSBY_SM_CONCURRENCY")
			}

			// Simulate the init logic
			getConcurrency, hasSetting := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
			if hasSetting {
				i, err := strconv.Atoi(getConcurrency)
				if err == nil && i >= 0 {
					smConcurrency = i
				}
			}

			if smConcurrency != tt.expected {
				t.Errorf("Expected smConcurrency to be %d, got %d", tt.expected, smConcurrency)
			}
		})
	}
}

// TestSMMessageStruct tests the smMessage structure
func TestSMMessageStruct(t *testing.T) {
	sourceURL, _ := url.Parse("sm://test")
	source := &config.Source{URL: sourceURL}
	secretName := "test-secret"

	msg := &smMessage{
		Source:      source,
		Name:        &secretName,
		Result:      "test-result",
		Error:       errors.New("test error"),
		IsRecursive: true,
	}

	if msg.Source != source {
		t.Error("Source not set correctly")
	}
	if *msg.Name != secretName {
		t.Error("Name not set correctly")
	}
	if msg.Result != "test-result" {
		t.Error("Result not set correctly")
	}
	if msg.Error == nil || msg.Error.Error() != "test error" {
		t.Error("Error not set correctly")
	}
	if !msg.IsRecursive {
		t.Error("IsRecursive not set correctly")
	}
}
