package main

import (
	"bytes"
	"fmt"
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

func env(m map[string]string) string {
	var buffer bytes.Buffer
	for k, v := range m {
		buffer.WriteString(fmt.Sprintf("export %s=\"%s\"", k, v))
		buffer.WriteString("\n")
	}
	return buffer.String()
}
