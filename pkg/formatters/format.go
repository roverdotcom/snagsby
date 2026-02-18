package formatters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"sort"
)

type formatterFunc func(map[string]string) string

// Formatters is a map of available formatters
var Formatters = map[string]formatterFunc{
	"env":     EnvFormater,
	"envfile": EnvFileFormater,
	"json":    JSONFormater,
}

// Merge updates the first map with values from the second
func Merge(i []map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range i {
		maps.Copy(out, m)
	}
	return out
}

func envEscape(i string) string {
	var envQuoteRegex = regexp.MustCompile(`(\\|"|\$|` + "`)")
	return envQuoteRegex.ReplaceAllString(i, `\$1`)
}

// EnvFormater returns a string to be evaluated by a shell for the setting of environment
// variables. The variables will be ordered by key
func EnvFormater(m map[string]string) string {
	var buffer bytes.Buffer
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	// Sort the keys for predictable export order
	sort.Strings(keys)
	for _, k := range keys {
		buffer.WriteString(fmt.Sprintf("export %s=\"%s\"", k, envEscape(m[k])))
		buffer.WriteString("\n")
	}

	return buffer.String()
}

// EnvFileFormater is similar to EnvFormater without the leading export
// declarations. This format can easily be piped to a file and loaded by a shell
// or systems like docker-compose.
func EnvFileFormater(m map[string]string) string {
	var buffer bytes.Buffer
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	// Sort the keys for predictable export order
	sort.Strings(keys)
	for _, k := range keys {
		buffer.WriteString(fmt.Sprintf("%s=\"%s\"", k, envEscape(m[k])))
		buffer.WriteString("\n")
	}

	return buffer.String()
}

// JSONFormater return a json representation of the map
func JSONFormater(m map[string]string) string {
	out, err := json.Marshal(m)
	if err != nil {
		return `{}`
	}
	return string(out)
}
