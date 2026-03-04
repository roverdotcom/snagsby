package resolvers

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
	connectortesting "github.com/roverdotcom/snagsby/pkg/connectors/testing"
)

func TestGetFilePath(t *testing.T) {
	examples := []struct {
		name         string
		urlString    string
		expectedPath string
	}{
		{
			name:         "absolute path with file scheme",
			urlString:    "file:///absolute/path/to/file.env",
			expectedPath: "/absolute/path/to/file.env",
		},
		{
			name:         "relative path with current directory",
			urlString:    "file://./pre-cache.env",
			expectedPath: "./pre-cache.env",
		},
		{
			name:         "relative path with parent directory",
			urlString:    "file://../parent/file.env",
			expectedPath: "../parent/file.env",
		},
		{
			name:         "relative path with 2 levels up directory",
			urlString:    "file://../../parent/file.env",
			expectedPath: "../../parent/file.env",
		},
		{
			name:         "simple filename",
			urlString:    "file://local.env",
			expectedPath: "local.env",
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			parsedURL, err := url.Parse(example.urlString)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}
			source := &config.Source{URL: parsedURL}
			actualPath := getFilePath(source)
			if actualPath != example.expectedPath {
				t.Errorf("Expected path '%s' but got '%s' (URL.Host='%s', URL.Path='%s')",
					example.expectedPath, actualPath, parsedURL.Host, parsedURL.Path)
			}
		})
	}
}

func TestProcessLine(t *testing.T) {
	examples := []struct {
		name          string
		line          string
		expectedKey   string
		expectedValue string
		expectedError error
	}{
		{
			name:          "empty line",
			line:          "",
			expectedKey:   "",
			expectedValue: "",
			expectedError: nil,
		},
		{
			name:          "comment line",
			line:          "# This is a comment",
			expectedKey:   "",
			expectedValue: "",
			expectedError: nil,
		},
		{
			name:          "simple key value",
			line:          "FOO=bar",
			expectedKey:   "FOO",
			expectedValue: "bar",
			expectedError: nil,
		},
		{
			name:          "key value with spaces",
			line:          " FOO = bar ",
			expectedKey:   "FOO",
			expectedValue: "bar",
			expectedError: nil,
		},
		{
			name:          "key value with inline comment",
			line:          "FOO=bar # This is a comment",
			expectedKey:   "FOO",
			expectedValue: "bar",
			expectedError: nil,
		},
		{
			name:          "invalid line without equals",
			line:          "FOO bar",
			expectedKey:   "",
			expectedValue: "",
			expectedError: fmt.Errorf("line 0: missing '=' separator"),
		},
		{
			name:          "empty key with value",
			line:          "=value",
			expectedKey:   "",
			expectedValue: "",
			expectedError: fmt.Errorf("line 0: empty key"),
		},
		{
			name:          "whitespace key with value",
			line:          "  =value",
			expectedKey:   "",
			expectedValue: "",
			expectedError: fmt.Errorf("line 0: empty key"),
		},
		{
			name:          "key with dash should fail",
			line:          "foo-bar=value",
			expectedKey:   "",
			expectedValue: "",
			expectedError: fmt.Errorf("invalid key 'foo-bar': environment variable names must contain only letters, digits, and underscores, and must start with a letter or underscore"),
		},
		{
			name:          "key with dot should fail",
			line:          "foo.bar=value",
			expectedKey:   "",
			expectedValue: "",
			expectedError: fmt.Errorf("invalid key 'foo.bar': environment variable names must contain only letters, digits, and underscores, and must start with a letter or underscore"),
		},
		{
			name:          "key starting with digit should fail",
			line:          "123FOO=value",
			expectedKey:   "",
			expectedValue: "",
			expectedError: fmt.Errorf("invalid key '123FOO': environment variable names must contain only letters, digits, and underscores, and must start with a letter or underscore"),
		},
		{
			name:          "valid key with underscore",
			line:          "FOO_BAR=value",
			expectedKey:   "FOO_BAR",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name:          "valid key lowercase",
			line:          "foo_bar=value",
			expectedKey:   "foo_bar",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name:          "valid key starting with underscore",
			line:          "_FOO=value",
			expectedKey:   "_FOO",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name:          "valid key with digits",
			line:          "FOO_123_BAR=value",
			expectedKey:   "FOO_123_BAR",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name:          "valid key with hash in value",
			line:          "FOO_123_BAR=\"value#withhash\"",
			expectedKey:   "FOO_123_BAR",
			expectedValue: "value#withhash",
			expectedError: nil,
		},
		{
			name:          "valid key with empty space / hash in value",
			line:          "FOO_123_BAR=\"value #withhash\"",
			expectedKey:   "FOO_123_BAR",
			expectedValue: "value #withhash",
			expectedError: nil,
		},
		{
			name:          "valid key with uneven quotes",
			line:          "FOO_123_BAR=\"value",
			expectedKey:   "FOO_123_BAR",
			expectedValue: "",
			expectedError: fmt.Errorf("line 0: uneven quotes for key 'FOO_123_BAR'"),
		},
		{
			name:          "empty quoted value",
			line:          "KEY=\"\"",
			expectedKey:   "KEY",
			expectedValue: "",
			expectedError: nil,
		},
		{
			name:          "single quoted value",
			line:          "KEY='value'",
			expectedKey:   "KEY",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name:          "single quotes with hash",
			line:          "KEY='value # with hash'",
			expectedKey:   "KEY",
			expectedValue: "value # with hash",
			expectedError: nil,
		},
		{
			name:          "uneven single quotes",
			line:          "KEY='value",
			expectedKey:   "KEY",
			expectedValue: "",
			expectedError: fmt.Errorf("line 0: uneven quotes for key 'KEY'"),
		},
		{
			name:          "value with equals sign",
			line:          "KEY=value=another",
			expectedKey:   "KEY",
			expectedValue: "value=another",
			expectedError: nil,
		},
		{
			name:          "quoted value with leading/trailing spaces",
			line:          "KEY=\"  value  \"",
			expectedKey:   "KEY",
			expectedValue: "  value  ",
			expectedError: nil,
		},
		{
			name:          "empty unquoted value",
			line:          "KEY=",
			expectedKey:   "KEY",
			expectedValue: "",
			expectedError: nil,
		},
		{
			name:          "quoted value with inline comment after",
			line:          "KEY=\"value\" # comment",
			expectedKey:   "KEY",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name:          "mixed quotes should fail",
			line:          "KEY=\"value'",
			expectedKey:   "KEY",
			expectedValue: "",
			expectedError: fmt.Errorf("line 0: uneven quotes for key 'KEY'"),
		},
		{
			name:          "value with newline in quotes",
			line:          "KEY=\"line1\\nline2\"",
			expectedKey:   "KEY",
			expectedValue: "line1\\nline2",
			expectedError: nil,
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			key, value, err := parseEnvLine(example.line, 0)
			if key != example.expectedKey {
				t.Errorf("Expected key '%s' but got '%s'", example.expectedKey, key)
			}
			if value != example.expectedValue {
				t.Errorf("Expected value '%s' but got '%s'", example.expectedValue, value)
			}
			if (err == nil && example.expectedError != nil) || (err != nil && example.expectedError == nil) || (err != nil && example.expectedError != nil && err.Error() != example.expectedError.Error()) {
				t.Errorf("Expected error '%v' but got '%v'", example.expectedError, err)
			}
		})
	}
}

func TestEnvFileResolve(t *testing.T) {
	// Create examples of env files and expected contents / errors
	examples := []struct {
		name                     string
		fileContents             string
		expectedItems            map[string]string
		expectedErrors           []string
		expectedSecretsRequested []string
	}{
		{
			name:                     "empty env file",
			fileContents:             "",
			expectedItems:            map[string]string{},
			expectedErrors:           []string{},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "simple env var",
			fileContents:             "FOO=bar",
			expectedItems:            map[string]string{"FOO": "bar"},
			expectedErrors:           []string{},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "simple env var with empty lines",
			fileContents:             "\nFOO=bar\n",
			expectedItems:            map[string]string{"FOO": "bar"},
			expectedErrors:           []string{},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "simple env var with comments",
			fileContents:             "#This is a comment\nFOO=bar # This is another comment\n",
			expectedItems:            map[string]string{"FOO": "bar"},
			expectedErrors:           []string{},
			expectedSecretsRequested: []string{},
		},

		{
			name:                     "simple env var in sm",
			fileContents:             "FOO=sm://path/to/secret",
			expectedItems:            map[string]string{"FOO": "resolved-value-for-sm://path/to/secret"},
			expectedErrors:           []string{},
			expectedSecretsRequested: []string{"path/to/secret"},
		},
		{
			name:                     "simple env var not found in sm",
			fileContents:             "FOO=sm://path/to/not-found",
			expectedItems:            map[string]string{},
			expectedErrors:           []string{"secret not found: sm://path/to/not-found"},
			expectedSecretsRequested: []string{"path/to/not-found"},
		},
		{
			name:                     "invalid line on line 3 reports correct line number",
			fileContents:             "FOO=bar\nBAR=baz\nno_equals_here\nQUX=quux",
			expectedItems:            map[string]string{"FOO": "bar", "BAR": "baz", "QUX": "quux"},
			expectedErrors:           []string{"line 3: missing '=' separator"},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "uneven quotes on line 2 reports correct line number",
			fileContents:             "FOO=bar\nBAR=\"unclosed",
			expectedItems:            map[string]string{"FOO": "bar"},
			expectedErrors:           []string{"line 2: uneven quotes for key 'BAR'"},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "empty key on line 1 reports correct line number",
			fileContents:             "=value\nFOO=bar",
			expectedItems:            map[string]string{"FOO": "bar"},
			expectedErrors:           []string{"line 1: empty key"},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "duplicate env var should return error",
			fileContents:             "FOO=bar\nFOO=baz",
			expectedItems:            map[string]string{"FOO": "bar"},
			expectedErrors:           []string{"duplicate key 'FOO' found in env file, duplicate keys are not supported"},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "key with dash should return validation error",
			fileContents:             "foo-bar=value1",
			expectedItems:            map[string]string{},
			expectedErrors:           []string{"invalid key 'foo-bar': environment variable names must contain only letters, digits, and underscores, and must start with a letter or underscore"},
			expectedSecretsRequested: []string{},
		},
		{
			name:                     "duplicate keys with same format should error",
			fileContents:             "MY_KEY=value1\nMY_KEY=value2",
			expectedItems:            map[string]string{"MY_KEY": "value1"},
			expectedErrors:           []string{"duplicate key 'MY_KEY' found in env file, duplicate keys are not supported"},
			expectedSecretsRequested: []string{},
		},
		{
			name: "multiple env vars pointing to same secret should dedupe API calls",
			fileContents: `FOO=sm://shared/secret
BAR=sm://shared/secret
BAZ=sm://other/secret
QUX=sm://shared/secret`,
			expectedItems: map[string]string{
				"FOO": "resolved-value-for-sm://shared/secret",
				"BAR": "resolved-value-for-sm://shared/secret",
				"BAZ": "resolved-value-for-sm://other/secret",
				"QUX": "resolved-value-for-sm://shared/secret",
			},
			expectedErrors: []string{},
			// Should only request each unique secret path once
			expectedSecretsRequested: []string{"shared/secret", "other/secret"},
		},
		{
			name:                     "keys are preserved exactly without normalization",
			fileContents:             "lowercase_var=value1\nMIXED_Case_Var=value2\nUPPERCASE_VAR=value3",
			expectedItems:            map[string]string{"lowercase_var": "value1", "MIXED_Case_Var": "value2", "UPPERCASE_VAR": "value3"},
			expectedErrors:           []string{},
			expectedSecretsRequested: []string{},
		},
	}

	requestedSecrets := []string{}
	mockSecretsManagerConnector := &connectortesting.MockSecretsConnector{
		GetSecretsFunc: func(keys []string) (map[string]string, []error) {
			requestedSecrets = append(requestedSecrets, keys...)
			secrets := make(map[string]string)
			errors := []error{}

			for _, key := range keys {
				if strings.Contains(key, "not-found") {
					errors = append(errors, fmt.Errorf("secret not found: sm://%s", key))
				} else {
					// Note: keys come without the "sm://" prefix as the resolver strips it
					secrets[key] = "resolved-value-for-sm://" + key
				}
			}
			return secrets, errors
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			requestedSecrets = []string{} // reset before each test
			// Create string readers and pass them to resolve function so tests are faster
			result := &Result{}
			envFileResolver := &EnvFileResolver{
				connector: mockSecretsManagerConnector,
			}
			envFileResolver.resolve(strings.NewReader(example.fileContents), result)

			// Check that the number of items matches
			if len(result.Items) != len(example.expectedItems) {
				t.Errorf("Expected %d items but got %d", len(example.expectedItems), len(result.Items))
			}

			// Check that expected items are present in the result
			for key, value := range example.expectedItems {
				actualValue, ok := result.Items[key]
				if !ok {
					t.Errorf("Expected item %s not found in result", key)
				} else if actualValue != value {
					t.Errorf("Expected item %s to have value %s but got %s", key, value, actualValue)
				}
			}

			// Check that non expected items are not present in the result
			for key := range result.Items {
				if _, ok := example.expectedItems[key]; !ok {
					t.Errorf("Unexpected item %s found in result", key)
				}
			}

			// Check that the number of errors matches
			if len(result.Errors) != len(example.expectedErrors) {
				t.Errorf("Expected %d errors but got %d", len(example.expectedErrors), len(result.Errors))
			}

			// Check that expected errors are present in the result
			for _, expectedError := range example.expectedErrors {
				found := false
				for _, err := range result.Errors {
					if err.Error() == expectedError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error '%s' not found in result errors", expectedError)
				}
			}

			// Check that the secrets requested match expectations
			if len(requestedSecrets) != len(example.expectedSecretsRequested) {
				t.Errorf("Expected %d secrets to be requested but got %d: %v", len(example.expectedSecretsRequested), len(requestedSecrets), requestedSecrets)
			}

			// Create maps for easier comparison (order doesn't matter)
			requestedMap := make(map[string]bool)
			for _, secret := range requestedSecrets {
				requestedMap[secret] = true
			}
			expectedMap := make(map[string]bool)
			for _, secret := range example.expectedSecretsRequested {
				expectedMap[secret] = true
			}

			// Check that all expected secrets were requested
			for secret := range expectedMap {
				if !requestedMap[secret] {
					t.Errorf("Expected secret '%s' to be requested but it wasn't", secret)
				}
			}

			// Check that no unexpected secrets were requested
			for secret := range requestedMap {
				if !expectedMap[secret] {
					t.Errorf("Unexpected secret '%s' was requested", secret)
				}
			}
		})
	}

}

// Using actual tmp files to test the full feature

const envFileContents = `# This is a comment
FOO=bar
BAZ=sm://path/to/baz
TEST=12345 # Inline comment
`

func TestEnvFileIntegrationTest(t *testing.T) {
	// Create a temporary .env file
	tmpFile, err := os.CreateTemp("", "envfile-test-*.env")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(envFileContents)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create the source configuration
	source := &config.Source{
		URL: &url.URL{
			Scheme: "sm",
			Host:   "us-east-1",
			Path:   "/",
		},
	}

	// Create the SecretsManagerConnector with fake secrets for testing
	secretsManagerConnector := connectortesting.NewSecretsManagerConnectorWithFakeSecrets(
		map[string]string{
			"path/to/baz": "secret-from-aws",
		},
		source,
	)

	// Create the real EnvFileResolver with the real connector
	envFileResolver := NewEnvFileResolver(secretsManagerConnector)

	// Create file source for the resolver
	fileSource := &config.Source{
		URL: &url.URL{
			Scheme: "file",
			Path:   tmpFile.Name(),
		},
	}

	// Resolve the env file (this tests the full integration)
	result := envFileResolver.Resolve(fileSource)

	// Verify results
	expectedItems := map[string]string{
		"FOO":  "bar",
		"BAZ":  "secret-from-aws",
		"TEST": "12345",
	}

	if len(result.Items) != len(expectedItems) {
		t.Errorf("Expected %d items but got %d", len(expectedItems), len(result.Items))
	}

	for key, value := range expectedItems {
		actualValue, ok := result.Items[key]
		if !ok {
			t.Errorf("Expected item %s not found in result", key)
		} else if actualValue != value {
			t.Errorf("Expected item %s to have value %s but got %s", key, value, actualValue)
		}
	}

	for key := range result.Items {
		if _, ok := expectedItems[key]; !ok {
			t.Errorf("Unexpected item %s found in result", key)
		}
	}

	// Should have no errors for this happy path test
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(result.Errors), result.Errors)
	}
}

// TestEnvFileIntegrationTestWithMissingSecret tests the integration when a secret is not found in AWS
func TestEnvFileIntegrationTestWithMissingSecret(t *testing.T) {
	// Create a temporary .env file with a reference to a non-existent secret
	tmpFile, err := os.CreateTemp("", "envfile-test-*.env")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := `FOO=bar
BAZ=sm://path/to/missing-secret
TEST=12345
`
	_, err = tmpFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create the source configuration
	source := &config.Source{
		URL: &url.URL{
			Scheme: "sm",
			Host:   "us-east-1",
			Path:   "/",
		},
	}

	// Create the SecretsManagerConnector with no fake secrets for testing
	secretsManagerConnector := connectortesting.NewSecretsManagerConnectorWithFakeSecrets(
		map[string]string{},
		source,
	)

	// Create the real EnvFileResolver with the real connector
	envFileResolver := NewEnvFileResolver(secretsManagerConnector)

	// Create file source for the resolver
	fileSource := &config.Source{
		URL: &url.URL{
			Scheme: "file",
			Path:   tmpFile.Name(),
		},
	}

	// Resolve the env file
	result := envFileResolver.Resolve(fileSource)

	// Verify that only non-secret items are present
	expectedItems := map[string]string{
		"FOO":  "bar",
		"TEST": "12345",
	}

	if len(result.Items) != len(expectedItems) {
		t.Errorf("Expected %d items but got %d", len(expectedItems), len(result.Items))
	}

	for key, value := range expectedItems {
		actualValue, ok := result.Items[key]
		if !ok {
			t.Errorf("Expected item %s not found in result", key)
		} else if actualValue != value {
			t.Errorf("Expected item %s to have value %s but got %s", key, value, actualValue)
		}
	}

	// Should have one error for the missing secret
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error but got %d", len(result.Errors))
	} else {
		errorMsg := result.Errors[0].Error()
		if !strings.Contains(errorMsg, "path/to/missing-secret") && !strings.Contains(errorMsg, "not found") {
			t.Errorf("Expected error message to mention missing secret, got: %s", errorMsg)
		}
	}
}
