package resolvers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roverdotcom/snagsby/pkg/config"
)

// KeyRegexp is the regular expression that keys must adhere to
var KeyRegexp *regexp.Regexp = regexp.MustCompile(`[^\w]`)

// Resolver defines an interface capable of resolving a Source to Result
type Resolver interface {
	Resolve(*config.Source) *Result
}

// Result stores a resolved result
type Result struct {
	Source *config.Source
	Errors []error
	Items  map[string]string
}

// AppendItem adds an item to the internal Items map
func (r *Result) AppendItem(key, value string) {
	// Initialize if we have to
	if r.Items == nil {
		r.Items = map[string]string{}
	}
	key = strings.ToUpper(KeyRegexp.ReplaceAllString(key, "_"))
	r.Items[key] = value
}

// AppendItems adds a map of items to our internal items store
func (r *Result) AppendItems(items map[string]string) {
	// Initialize if we have to
	if r.Items == nil {
		r.Items = map[string]string{}
	}
	for k, v := range items {
		r.AppendItem(k, v)
	}
}

// AppendError adds an error to the result
func (r *Result) AppendError(err error) {
	r.Errors = append(r.Errors, err)
}

// HasErrors indicates whether or not this result has errors
func (r *Result) HasErrors() bool {
	return len(r.Errors) > 0
}

// ItemKeys returns the item key names
func (r *Result) ItemKeys() []string {
	keys := make([]string, 0, len(r.Items))
	for k := range r.Items {
		keys = append(keys, k)
	}
	return keys
}

// LenItems returns the number of Items stored
func (r *Result) LenItems() int {
	return len(r.Items)
}

// ResolveSource will resolve a config.Source to a Result object
func ResolveSource(source *config.Source) *Result {
	sourceURL := source.URL
	var s Resolver
	if sourceURL.Scheme == "sm" {
		s = &SecretsManagerResolver{}
	} else if sourceURL.Scheme == "s3" {
		s = &S3ManagerResolver{}
	} else if sourceURL.Scheme == "manifest" {
		s = &ManifestResolver{}
	} else {
		return &Result{Source: source, Errors: []error{fmt.Errorf("No resolver found for scheme %s", sourceURL.Scheme)}}
	}

	return s.Resolve(source)
}
