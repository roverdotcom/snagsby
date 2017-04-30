package main

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

	if _, err := config.SetSources(emptyArgs, emptyEnv); err == nil {
		t.Errorf("Expected an error parsing empty args and env")
	}
	if o, _ := config.SetSources(sourcesArgs, ""); o[0].Host != "bucket" {
		t.Errorf("Host is actually %s", o[0].Host)
	}
	if o, _ := config.SetSources(emptyArgs, sourcesEnv); o[1].Path != "/two.json" {
		t.Errorf("Path is actually %s", o[1].Path)
	}
}
