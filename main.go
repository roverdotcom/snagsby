package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/roverdotcom/snagsby/secrets"
)

var (
	showVersion = false
	setFail     = false
)

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

	ch := make(chan *secrets.Collection, config.LenSources())
	for _, source := range config.sources {
		go func(s *url.URL) {
			ch <- secrets.LoadSecretsFromSource(s)
		}(source)
	}

	for i := 0; i < config.LenSources(); i++ {
		col := <-ch

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

		col.Print()
	}
}
