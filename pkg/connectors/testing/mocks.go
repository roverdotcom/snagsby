package testing

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
