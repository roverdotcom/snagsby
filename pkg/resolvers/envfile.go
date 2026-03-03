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

// parseEnvLine parses a single line from an env file into a key-value pair.
// It validates env var names, strips comments and whitespace, and handles quoted values.
// Quoted values (using " or ') preserve their content including hashes and whitespace.
// Returns empty strings for blank lines or comment-only lines (no error).
func parseEnvLine(line string) (string, string, error) {
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

	// Handle quoted values (both single and double quotes)
	if len(value) >= 1 {
		firstChar := value[0]
		if firstChar == '"' || firstChar == '\'' {
			// Find the closing quote
			closingQuoteIdx := strings.IndexByte(value[1:], firstChar)
			if closingQuoteIdx == -1 {
				// No closing quote found
				return key, "", fmt.Errorf("invalid line: %s (uneven quotes)", line)
			}
			// closingQuoteIdx is relative to value[1:], so actual index is closingQuoteIdx + 1
			actualClosingIdx := closingQuoteIdx + 1
			// Extract the value between quotes
			quotedValue := value[1:actualClosingIdx]
			// Everything after the closing quote is ignored (including comments)
			return key, quotedValue, nil
		}
	}

	// Remove inline comments for unquoted values
	if idx := strings.Index(value, " #"); idx != -1 {
		value = strings.TrimSpace(value[:idx])
	}
	return key, value, nil
}

// parsedEnvFile holds the parsed environment variables from a file.
type parsedEnvFile struct {
	envVars         map[string]string
	envVarsOrder    []string
	needsResolution map[string]string
}

// parseEnvFile reads and parses an env file, identifying variables and secrets.
// Returns parsed data structure with env vars, their order, and secrets needing resolution.
func parseEnvFile(file io.Reader, result *Result) *parsedEnvFile {
	parsed := &parsedEnvFile{
		envVars:         make(map[string]string),
		envVarsOrder:    []string{},
		needsResolution: map[string]string{},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		key, value, err := parseEnvLine(line)
		if err != nil {
			result.AppendError(err)
			continue
		}

		if key == "" {
			continue
		}

		// Check for duplicate keys
		if _, exists := parsed.envVars[key]; exists {
			result.AppendError(fmt.Errorf("duplicate key '%s' found in env file, duplicate keys are not supported", key))
			continue
		}

		parsed.envVars[key] = value
		parsed.envVarsOrder = append(parsed.envVarsOrder, key)

		// If the value points to sm, we will need to resolve it before we can add it to the result
		if strings.HasPrefix(value, "sm://") {
			parsed.needsResolution[key] = strings.TrimPrefix(value, "sm://")
		}
	}
	if err := scanner.Err(); err != nil {
		result.AppendError(err)
	}

	return parsed
}

// populateResultWithSecrets adds environment variables to the result, resolving secrets as needed.
func populateResultWithSecrets(parsed *parsedEnvFile, secrets map[string]string, result *Result) {
	for _, key := range parsed.envVarsOrder {
		secretPath, needsSecret := parsed.needsResolution[key]
		if needsSecret {
			if secretValue, found := secrets[secretPath]; found {
				result.AppendItemExact(key, secretValue)
			}
			// If secret not found, skip it (error already reported during GetSecrets)
		} else {
			result.AppendItemExact(key, parsed.envVars[key])
		}
	}
}

func (e *EnvFileResolver) resolve(file io.Reader, result *Result) {
	parsed := parseEnvFile(file, result)

	// All lines have explicit values. No need to resolve them.
	if len(parsed.needsResolution) == 0 {
		for _, key := range parsed.envVarsOrder {
			result.AppendItemExact(key, parsed.envVars[key])
		}
		return
	}

	// The values in the original env file contain the path for secrets manager
	// Dedupe secret keys to avoid redundant API calls
	secretKeysMap := make(map[string]bool)
	for _, secretPath := range parsed.needsResolution {
		secretKeysMap[secretPath] = true
	}
	secretKeys := slices.Collect(maps.Keys(secretKeysMap))
	secrets, errors := e.connector.GetSecrets(secretKeys)

	for _, err := range errors {
		result.AppendError(err)
	}

	populateResultWithSecrets(parsed, secrets, result)
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
