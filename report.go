package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Report is used to generate a report of what keys where loaded from where and
// which ones overrode others.
type Report struct {
	Collections []*Collection
}

// AppendCollection will add to the internal collection
func (r *Report) AppendCollection(c *Collection) {
	r.Collections = append(r.Collections, c)
}

// Generate returns a report string
func (r *Report) Generate() string {
	var out bytes.Buffer
	keyMap := r.getKeyMap()
	out.WriteString("Snagsby:\n")
	for _, res := range r.Collections {
		var keys []string
		for _, key := range res.Keys() {
			var overrideIndicator string
			if len(keyMap[key]) > 1 {
				if keyMap[key][len(keyMap[key])-1] == res.Source {
					overrideIndicator = "+"
				} else {
					overrideIndicator = "-"
				}

			}
			keys = append(keys, overrideIndicator+key)
		}
		sort.Strings(keys)
		out.WriteString(fmt.Sprintf("\t%s: (%s)\n", res.Source, strings.Join(keys[:], ", ")))
	}
	return out.String()
}

func (r *Report) getKeyMap() map[string][]string {
	out := map[string][]string{}
	for _, res := range r.Collections {
		for _, key := range res.Keys() {
			if _, ok := out[key]; !ok {
				var s []string
				out[key] = s
			}
			out[key] = append(out[key], res.Source)
		}
	}
	return out
}
