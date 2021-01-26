package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/roverdotcom/snagsby/pkg"
	"github.com/roverdotcom/snagsby/pkg/app"
	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/formatters"
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
		fmt.Printf("snagsby version %s (aws sdk: %s golang: %s)\n", pkg.Version, aws.SDKVersion, runtime.Version())
		return
	}

	// Make sure we were given a valid formatter
	formatter, ok := formatters.Formatters[format]
	if !ok {
		fmt.Fprintln(os.Stderr, "No formatter found")
		os.Exit(2)
	}

	snagsbyConfig := config.NewConfig()
	err := snagsbyConfig.SetSources(flagSet.Args(), os.Getenv("SNAGSBY_SOURCE"))
	if err != nil {
		fmt.Printf("Error parsing sources: %s\n", err)
		os.Exit(1)
	}

	results := app.ResolveConfigSources(snagsbyConfig)
	var resultsMap []map[string]string
	for _, result := range results {
		if result.HasErrors() {
			// Print errors to stderr
			fmt.Fprintln(os.Stderr, "Error processing snagsby source:", result.Source.URL.String())
			for _, err := range result.Errors {
				fmt.Fprintln(os.Stderr, err)
			}

			// Bail if we're exiting on failure
			if setFail {
				os.Exit(1)
			}

			continue
		}

		resultsMap = append(resultsMap, result.Items)
	}

	// Merge together our rendered sources which are listed in the order they
	// were specified.
	all := formatters.Merge(resultsMap)
	fmt.Print(formatter(all))
}
