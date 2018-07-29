package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
)

// SourceReport holds report information for each individual snagsby source
type SourceReport struct {
	Source string
	Keys   []string
}

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

	// Header Row
	tabWriter.Write([]byte("Snagsby Keys\tSnagsby Source\n"))

	sourceReports := r.getSourceReports()

	for _, sourceReport := range sourceReports {
		fmt.Fprintln(tabWriter, fmt.Sprintf(
			"%s\t%s", strings.Join(sourceReport.Keys[:], ", "), sourceReport.Source))
	}

	tabWriter.Flush()
	return out.String()
}

// getSourceReports returns a list of source reports
func (r *Report) getSourceReports() []*SourceReport {
	out := make([]*SourceReport, len(r.Collections))
	keyMap := r.getKeyMap()

	for idx, res := range r.Collections {
		sourceName := fmt.Sprintf("%d-%s", idx, res.Source)
		sourceReport := SourceReport{
			Source: res.Source,
			Keys:   make([]string, len(res.Keys())),
		}
		out[idx] = &sourceReport

		for idx, key := range res.Keys() {
			var overrideIndicator string
			if len(keyMap[key]) > 1 {
				if keyMap[key][len(keyMap[key])-1] == sourceName {
					overrideIndicator = "+"
				} else {
					overrideIndicator = "-"
				}

			}
			sourceReport.Keys[idx] = overrideIndicator + key
		}
	}

	return out
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
