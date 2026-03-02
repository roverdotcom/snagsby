package config

import (
	"net/url"
	"regexp"
	"strings"
)

var commaSplit = regexp.MustCompile(`[\s|,]+`)

// Source represents a single snagsby source URI
type Source struct {
	URL *url.URL
}

func splitEnvArg(envArg string) []string {
	return commaSplit.Split(strings.TrimSpace(envArg), -1)
}

// NewConfig returns a new configuration
func NewConfig() *Config {
	return &Config{}
}

// Config is the main configuration object
type Config struct {
	Sources []*Source
}

// SetSources will set the internal sources slice from a list of strings or
// from a single environment string
func (c *Config) SetSources(args []string, env string) error {
	var rawSources []string
	var sources []*Source

	// Re-initialize
	c.Sources = sources

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
		c.Sources = append(c.Sources, &Source{url})
	}

	return nil
}

// GetSources returns the internal sources
func (c *Config) GetSources() []*Source {
	return c.Sources
}

// lenSources is the number of sources
func (c *Config) lenSources() int {
	return len(c.Sources)
}
