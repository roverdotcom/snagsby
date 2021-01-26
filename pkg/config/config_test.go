package config

import (
	"testing"
)

func TestSplitEnvArg(t *testing.T) {
	o := splitEnvArg("Charles  ,  Dickens")
	if o[0] != "Charles" {
		t.Fail()
	}
	if o[1] != "Dickens" {
		t.Fail()
	}
	spacedString := splitEnvArg("Charles     Dickens")
	if spacedString[0] != "Charles" {
		t.Errorf("Expecting Charles got %s", spacedString)
	}

	if v := splitEnvArg(" charles  dickens"); v[0] != "charles" {
		t.Errorf("Expected charles got %s", v)
	}

	if v := splitEnvArg(" charles  "); v[0] != "charles" {
		t.Errorf("Expected charles got %s", v)
	}

	multiLine := splitEnvArg(`
	 s3://charles/dickens |

	    sm://oliver/twist
	`)

	if multiLine[0] != "s3://charles/dickers" &&
		multiLine[1] != "sm://oliver/twist" {
		t.Errorf("Error splitting with newlines %s", multiLine)
	}
}

func TestGetSources(t *testing.T) {
	emptyArgs := []string{}
	emptyEnv := ""
	sourcesArgs := []string{
		"s3://bucket/one.json",
		"s3://bucket/two.json",
	}
	sourcesEnv := "s3://bucket/one.json, s3://bucket/two.json"
	config := NewConfig()

	// Passing in no sources is fine
	config.SetSources(emptyArgs, emptyEnv)
	if config.LenSources() != 0 {
		t.Errorf("Expected an error parsing empty args and env")
	}

	if config.SetSources(sourcesArgs, ""); config.Sources[0].URL.Host != "bucket" {
		t.Errorf("Host is actually %s", config.Sources[0].URL.Host)
	}
	if config.SetSources(emptyArgs, sourcesEnv); config.Sources[1].URL.Path != "/two.json" {
		t.Errorf("Path is actually %s", config.Sources[1].URL.Path)
	}

	err := config.SetSources([]string{":"}, "")
	if err == nil || config.LenSources() != 0 {
		t.Errorf("Expected a parsing url for the : url")
	}

	err = config.SetSources([]string{}, `"sm://nicholas/nickleby, sm://esther/summerson`)
	if err == nil || config.LenSources() != 0 {
		t.Errorf("Expected a parsing url for the : url")
	}
}
