package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
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
	tabWriter := tabwriter.NewWriter(&out, 0, 8, 0, '\t', 0)
	keyMap := r.getKeyMap()

	// Header Row
	tabWriter.Write([]byte("Snagsby Keys\tSnagsby Source\n"))

	for idx, res := range r.Collections {
		sourceName := fmt.Sprintf("%d-%s", idx, res.Source)
		var keys []string
		for _, key := range res.Keys() {
			var overrideIndicator string
			if len(keyMap[key]) > 1 {
				if keyMap[key][len(keyMap[key])-1] == sourceName {
					overrideIndicator = "+"
				} else {
					overrideIndicator = "-"
				}

			}
			keys = append(keys, overrideIndicator+key)
		}
		fmt.Fprintln(tabWriter, fmt.Sprintf("%s\t%s", strings.Join(keys[:], ", "), res.Source))
	}
	tabWriter.Flush()
	return out.String()
}

func (r *Report) getKeyMap() map[string][]string {
	out := map[string][]string{}
	for idx, res := range r.Collections {
		sourceName := fmt.Sprintf("%d-%s", idx, res.Source)
		for _, key := range res.Keys() {
			if _, ok := out[key]; !ok {
				var s []string
				out[key] = s
			}
			out[key] = append(out[key], sourceName)
		}
	}
	return out
}
