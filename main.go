package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
)

var (
	showVersion = false
	setFail     = false
)

var format string

func main() {
	flagSet := flag.NewFlagSet("snagsby", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Example usage: snagsby s3://my-bucket/my-config.json?region=us-west-2\n")
		flagSet.PrintDefaults()
	}
	flagSet.BoolVar(&showVersion, "v", false, "print version string")
	flagSet.BoolVar(&setFail, "e", false, "fail on errors")
	flagSet.StringVar(&format, "o", "env", "Output")
	flagSet.StringVar(&format, "output", "env", "Output")
	flagSet.Parse(os.Args[1:])

	if showVersion {
		fmt.Printf("snagsby version %s (aws sdk: %s golang: %s)\n", Version, aws.SDKVersion, runtime.Version())
		return
	}

	// Make sure we were given a valid formatter
	formatter, ok := formatters[format]
	if !ok {
		fmt.Fprintln(os.Stderr, "No formatter found")
		os.Exit(2)
	}

	config := NewConfig()
	err := config.SetSources(flagSet.Args(), os.Getenv("SNAGSBY_SOURCE"))
	if err != nil {
		fmt.Printf("Error parsing sources: %s\n", err)
		os.Exit(1)
	}

	var jobs []chan *Collection
	for _, source := range config.sources {
		job := make(chan *Collection, 1)
		jobs = append(jobs, job)
		go func(s *url.URL, c chan *Collection) {
			job <- LoadItemsFromSource(s)
		}(source, job)
	}

	var rendered []map[string]string
	for _, result := range jobs {
		col := <-result

		if col.Error != nil {
			// Print errors to stderr
			fmt.Fprintln(os.Stderr, "Error processing snagsby source:", col.Source)
			fmt.Fprintln(os.Stderr, col.Error)

			// Bail if we're exiting on failure
			if setFail {
				os.Exit(1)
			}

			continue
		}

		rendered = append(rendered, col.AsMap())
	}

	// Merge together our rendered sources which are listed in the order they
	// were specified.
	all := merge(rendered)
	fmt.Print(formatter(all))
}
