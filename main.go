package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
)

var (
	showVersion = false
	setFail     = false
)

func merge(i []map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range i {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func main() {
	flagSet := flag.NewFlagSet("snagsby", flag.ExitOnError)
	flagSet.Usage = func() {
		// TODO: actual usage
		fmt.Fprintf(os.Stderr, "Usage of snagsby:\n")
		flagSet.PrintDefaults()
	}
	flagSet.BoolVar(&showVersion, "v", false, "print version string")
	flagSet.BoolVar(&setFail, "e", false, "fail on errors")
	flagSet.Parse(os.Args[1:])

	if showVersion {
		fmt.Printf("snagsby version %s (aws sdk: %s)\n", VERSION, aws.SDKVersion)
		return
	}

	config := NewConfig()
	config.SetSources(flagSet.Args(), os.Getenv("SNAGSBY_SOURCE"))

	var jobs []chan *Collection
	for _, source := range config.sources {
		job := make(chan *Collection)
		jobs = append(jobs, job)
		go func(s *url.URL, c chan *Collection) {
			job <- LoadSecretsFromSource(s)
		}(source, job)
	}

	var rendered []map[string]string
	for _, result := range jobs {
		col := <-result

		if col.Error != nil {
			// Print errors to stderr
			fmt.Fprintln(os.Stderr, "Error parsing:", col.Source)
			fmt.Fprintln(os.Stderr, col.Error)

			// Bail if we're exiting on failure
			if setFail {
				os.Exit(1)
			}

			continue
		}

		rendered = append(rendered, col.AsMap())
	}

	all := merge(rendered)
	fmt.Print(env(all))
}
