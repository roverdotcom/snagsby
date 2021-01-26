package app

import (
	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/resolvers"
)

// ResolveConfigSources resolves a source config out to results
func ResolveConfigSources(snagsbyConfig *config.Config) []*resolvers.Result {
	var jobs []chan *resolvers.Result
	var out []*resolvers.Result
	for _, source := range snagsbyConfig.GetSources() {
		job := make(chan *resolvers.Result, 1)
		jobs = append(jobs, job)
		go func(s *config.Source, c chan *resolvers.Result) {
			job <- resolvers.ResolveSource(s)
		}(source, job)
	}

	for _, job := range jobs {
		out = append(out, <-job)
	}

	return out
}
