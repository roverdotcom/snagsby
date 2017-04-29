package main

import (
	"bytes"
	"fmt"
)

func env(m map[string]string) string {
	var buffer bytes.Buffer
	for k, v := range m {
		buffer.WriteString(fmt.Sprintf("export %s\"%s\"", k, v))
		buffer.WriteString("\n")
	}
	return buffer.String()
}
