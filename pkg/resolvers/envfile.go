package resolvers

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/roverdotcom/snagsby/pkg/config"
)

// envVarNameRegexp validates environment variable names
// Must start with letter or underscore, followed by letters, digits, or underscores
var envVarNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type envFileSecretsGetter interface {
	GetSecrets(keys []string) (map[string]string, []error)
}

type EnvFileResolver struct {
	connector envFileSecretsGetter
}

func NewEnvFileResolver(connector envFileSecretsGetter) *EnvFileResolver {
	return &EnvFileResolver{connector: connector}
}

func isValidEnvVarName(key string) bool {
	return envVarNameRegexp.MatchString(key)
}

func getFilePath(source *config.Source) string {
	// For file:// URLs, if there's a host component (e.g., file://./path or file://filename),
	// we need to concatenate host and path to preserve relative paths
	if source.URL.Host != "" {
		// Handle case where path is empty (e.g., file://filename)
		if source.URL.Path == "" {
			return source.URL.Host
		}
		// Remove leading slash from path to avoid double slashes
		path := strings.TrimPrefix(source.URL.Path, "/")
		return fmt.Sprintf("%s/%s", source.URL.Host, path)
	}
	return source.URL.Path
}

func processLine(line string) (string, string, error) {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
		return "", "", nil
	}

	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid line: %s", line)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if key == "" {
		return "", "", fmt.Errorf("invalid line: %s (empty key)", line)
	}

	if !isValidEnvVarName(key) {
		return "", "", fmt.Errorf("invalid key '%s': environment variable names must contain only letters, digits, and underscores, and must start with a letter or underscore", key)
	}

	// Remove inline comments
	if idx := strings.Index(value, " #"); idx != -1 {
		value = strings.TrimSpace(value[:idx])
	}
	return key, value, nil
}

func (e *EnvFileResolver) resolve(file io.Reader, result *Result) {
	needsResolution := map[string]string{}

	envVars := make(map[string]string)

	// We will use this to ensure we keep the same order of env vars as in the file
	envVarsOrder := []string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		key, value, err := processLine(line)
		if err != nil {
			result.AppendError(err)
			continue
		}

		if key == "" {
			continue
		}

		// Check for duplicate keys
		if _, exists := envVars[key]; exists {
			result.AppendError(fmt.Errorf("duplicate key '%s' found in env file, duplicate keys are not supported", key))
			continue
		}

		envVars[key] = value
		envVarsOrder = append(envVarsOrder, key)

		// If the value points to sm, we will need to resolve it before we can add it to the result
		if strings.HasPrefix(value, "sm://") {
			needsResolution[key] = strings.TrimPrefix(value, "sm://")
		}
	}
	if err := scanner.Err(); err != nil {
		result.AppendError(err)
	}

	// All lines have explicit values. No need to resolve them.
	if len(needsResolution) == 0 {

		for _, key := range envVarsOrder {
			result.AppendItem(key, envVars[key])
		}
		return
	}

	// The values in the original env file contain the path for secrets manager
	// Dedupe secret keys to avoid redundant API calls
	secretKeysMap := make(map[string]bool)
	for _, secretPath := range needsResolution {
		secretKeysMap[secretPath] = true
	}
	secretKeys := slices.Collect(maps.Keys(secretKeysMap))
	secrets, errors := e.connector.GetSecrets(secretKeys)

	for _, err := range errors {
		result.AppendError(err)
	}

	for _, key := range envVarsOrder {
		if _, ok := needsResolution[key]; ok {
			if _, ok := secrets[needsResolution[key]]; ok {
				result.AppendItem(key, secrets[needsResolution[key]])
			}
		} else {
			result.AppendItem(key, envVars[key])
		}
	}

}

func (e *EnvFileResolver) Resolve(source *config.Source) *Result {
	result := &Result{Source: source}

	filePath := getFilePath(source)
	fileReader, err := os.Open(filePath)
	if err != nil {
		result.AppendError(err)
		return result
	}
	defer fileReader.Close()

	e.resolve(fileReader, result)

	return result
}
