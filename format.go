package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

func merge(i []map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range i {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// EnvFormat returns a string to be evaluated by a shell for the setting of environment
// variables. The variables will be ordered by key
func EnvFormat(m map[string]string) string {
	var buffer bytes.Buffer
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	// Sort the keys for predictable export order
	sort.Strings(keys)
	for _, k := range keys {
		buffer.WriteString(fmt.Sprintf("export %s=\"%s\"", k, m[k]))
		buffer.WriteString("\n")
	}

	return buffer.String()
}

// JSONFormat return a json representation of the map
func JSONFormat(m map[string]string) string {
	out, err := json.Marshal(m)
	if err != nil {
		return `{}`
	}
	return string(out)
}
