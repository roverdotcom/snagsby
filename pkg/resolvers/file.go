package resolvers

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/roverdotcom/snagsby/pkg/config"
)

type EnvFileItem struct {
	Key             string
	Value           string
	NeedsResolution bool
}

func parseEnvFile(filePath string) ([]*EnvFileItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []*EnvFileItem
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Skip if key is empty
		if key == "" {
			continue
		}

		// Remove surrounding quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
		                        (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		item := &EnvFileItem{
			Key:             key,
			Value:           value,
			NeedsResolution: strings.HasPrefix(value, "sm://"),
		}

		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func resolveEnvFileItems(items []*EnvFileItem, result *Result) {
	// Separate items that need AWS resolution from direct values
	var awsItems []*EnvFileItem
	secretIDs := []string{}
	secretIDToKey := make(map[string]string) // Map secretID -> env key

	for _, item := range items {
		if item.NeedsResolution {
			// Extract the secret ID from sm:// reference
			secretID := strings.TrimPrefix(item.Value, "sm://")
			awsItems = append(awsItems, item)
			secretIDs = append(secretIDs, secretID)
			secretIDToKey[secretID] = item.Key
		} else {
			// Direct value - add immediately
			result.AppendItem(item.Key, item.Value)
		}
	}

	// If no AWS items, we're done
	if len(awsItems) == 0 {
		return
	}

	// Fetch all secrets using shared batch function
	secretValues, errors := BatchFetchSecrets(secretIDs, 20)

	// Add errors to result
	for _, err := range errors {
		result.AppendError(err)
	}

	// Add fetched secrets to result
	for secretID, value := range secretValues {
		if key, ok := secretIDToKey[secretID]; ok {
			result.AppendItem(key, value)
		}
	}
}

type FileResolver struct{}

func (f *FileResolver) Resolve(source *config.Source) *Result {
	result := &Result{Source: source}

	// Construct file path from URL
	filePath := fmt.Sprintf("%s%s", source.URL.Host, source.URL.Path)

	// Parse the env file
	items, err := parseEnvFile(filePath)
	if err != nil {
		result.AppendError(err)
		return result
	}

	// Resolve items (both direct values and AWS secrets)
	resolveEnvFileItems(items, result)

	return result
}
