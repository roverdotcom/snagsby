package testing

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/connectors"
)

// MockSecretsConnector is a reusable mock for any connector that implements secret retrieval methods.
// It can be used by all resolvers (manifest, envfile, secretsmanager, etc.) that need to mock secret operations.
//
// Usage example:
//
//	mock := &testing.MockSecretsConnector{
//		GetSecretsFunc: func(keys []string) (map[string]string, []error) {
//			return map[string]string{"key1": "value1"}, nil
//		},
//		GetSecretFunc: func(secretName string) (string, error) {
//			return `{"key":"value"}`, nil
//		},
//		ListSecretsFunc: func(prefix string) ([]string, error) {
//			return []string{"secret1", "secret2"}, nil
//		},
//	}
type MockSecretsConnector struct {
	GetSecretsFunc  func(keys []string) (map[string]string, []error)
	GetSecretFunc   func(secretName string) (string, error)
	ListSecretsFunc func(prefix string) ([]string, error)
}

// GetSecrets retrieves multiple secrets by their keys.
func (m *MockSecretsConnector) GetSecrets(keys []string) (map[string]string, []error) {
	if m.GetSecretsFunc != nil {
		return m.GetSecretsFunc(keys)
	}
	return map[string]string{}, nil
}

// GetSecret retrieves a single secret by name.
func (m *MockSecretsConnector) GetSecret(secretName string) (string, error) {
	if m.GetSecretFunc != nil {
		return m.GetSecretFunc(secretName)
	}
	return "", nil
}

// ListSecrets lists all secrets with the given prefix.
func (m *MockSecretsConnector) ListSecrets(prefix string) ([]string, error) {
	if m.ListSecretsFunc != nil {
		return m.ListSecretsFunc(prefix)
	}
	return []string{}, nil
}

// MockSecretsManagerAPIClient is a mock implementation of the AWS Secrets Manager API client.
// This allows testing the full integration of Resolver + Connector with a mocked AWS SDK client.
//
// Usage example:
//
//	mock := &testing.MockSecretsManagerAPIClient{
//		Secrets: map[string]string{
//			"path/to/secret": "secret-value",
//		},
//	}
type MockSecretsManagerAPIClient struct {
	// Secrets maps secret names to their string values
	Secrets map[string]string

	// GetSecretValueFunc allows custom behavior for GetSecretValue
	GetSecretValueFunc func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)

	// ListSecretsFunc allows custom behavior for ListSecrets
	ListSecretsFunc func(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
}

// GetSecretValue implements the GetSecretValueAPIClient interface.
func (m *MockSecretsManagerAPIClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.GetSecretValueFunc != nil {
		return m.GetSecretValueFunc(ctx, params, optFns...)
	}

	if m.Secrets == nil {
		return nil, fmt.Errorf("ResourceNotFoundException: secret not found")
	}

	secretName := aws.ToString(params.SecretId)
	value, exists := m.Secrets[secretName]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: secret %s not found", secretName)
	}

	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(value),
	}, nil
}

// ListSecrets implements the ListSecretsAPIClient interface.
func (m *MockSecretsManagerAPIClient) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	if m.ListSecretsFunc != nil {
		return m.ListSecretsFunc(ctx, params, optFns...)
	}

	return &secretsmanager.ListSecretsOutput{
		SecretList: []types.SecretListEntry{},
	}, nil
}

// NewSecretsManagerConnectorWithFakeSecrets creates a SecretsManagerConnector with fake secrets for testing.
// This is a convenience function that encapsulates the mock setup so tests don't need to know about the mock client implementation.
//
// Usage example:
//
//	connector := testing.NewSecretsManagerConnectorWithFakeSecrets(
//		map[string]string{
//			"path/to/secret": "secret-value",
//		},
//		source,
//	)
func NewSecretsManagerConnectorWithFakeSecrets(secrets map[string]string, source *config.Source) *connectors.SecretsManagerConnector {
	mockClient := &MockSecretsManagerAPIClient{
		Secrets: secrets,
	}
	return connectors.NewSecretsManagerConnectorWithClient(mockClient, source)
}
