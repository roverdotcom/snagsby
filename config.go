package main

import (
	"errors"
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
func (c *Config) SetSources(args []string, env string) ([]*url.URL, error) {
	var sources []string
	var urls []*url.URL

	if len(args) == 0 && env == "" {
		return urls, errors.New("No source provided")
	}

	if len(args) > 0 {
		sources = args
	} else {
		sources = splitEnvArg(env)
	}

	for _, source := range sources {
		url, _ := url.Parse(source)
		urls = append(urls, url)
	}

	c.sources = urls
	return urls, nil
}

// LenSources is the number or sources
func (c *Config) LenSources() int {
	return len(c.sources)
}
