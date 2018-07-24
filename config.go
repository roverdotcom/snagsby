package main

import (
	"net/url"
	"regexp"
	"strings"
)

var commaSplit = regexp.MustCompile(`[\s|,]+`)

// Config is the main configuration object
type Config struct {
	sources []*url.URL
}

// NewConfig returns a new configuration
func NewConfig() *Config {
	return &Config{}
}

func splitEnvArg(envArg string) []string {
	return commaSplit.Split(strings.TrimSpace(envArg), -1)
}

// SetSources will set the internal sources slice from a list of strings or
// from a single environment string
func (c *Config) SetSources(args []string, env string) error {
	var rawSources []string
	var sources []*url.URL

	// Re-initialize
	c.sources = sources

	if len(args) == 0 && env == "" {
		return nil
	}

	if len(args) > 0 {
		rawSources = args
	} else {
		rawSources = splitEnvArg(env)
	}

	for _, rawSource := range rawSources {
		url, err := url.Parse(rawSource)
		if err != nil {
			return err
		}
		c.sources = append(c.sources, url)
	}

	return nil
}

// LenSources is the number or sources
func (c *Config) LenSources() int {
	return len(c.sources)
}
