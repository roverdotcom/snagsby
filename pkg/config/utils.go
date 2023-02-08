package config

import (
	"os"
	"regexp"
)

// Case insensitive 1, yes, or true strings from an
// environment variable
var strBool = regexp.MustCompile(`(?i)^(1|true|yes)$`)

func EnvBool(envName string) bool {
	env := os.Getenv(envName)
	return strBool.Match([]byte(env))
}
