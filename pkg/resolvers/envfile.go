package resolvers

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/roverdotcom/snagsby/pkg/config"
)

type envFileSecretsGetter interface {
	GetSecrets(keys []string) (map[string]string, []error)
}

type EnvFileResolver struct {
	connector envFileSecretsGetter
}

func NewEnvFileResolver(connector envFileSecretsGetter) *EnvFileResolver {
	return &EnvFileResolver{connector: connector}
}

func getFilePath(source *config.Source) string {
	if source.URL.Scheme == "file" {
		return source.URL.Path
	}
	return fmt.Sprintf("%s%s", source.URL.Host, source.URL.Path)
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
	secretKeys := slices.Collect(maps.Values(needsResolution))
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
