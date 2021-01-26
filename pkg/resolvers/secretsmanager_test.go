package resolvers

import (
	"net/url"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
)

func TestIsRecursive(t *testing.T) {
	sm := &SecretsManagerResolver{}
	var url *url.URL
	var res bool
	var err error

	url, err = url.Parse("sm://hello/world/*")
	res = sm.isRecursive(&config.Source{URL: url})
	if res != true || err != nil {
		t.Errorf("Is recursive failed test %s", err)
	}

	url, err = url.Parse("sm://hello/world")
	res = sm.isRecursive(&config.Source{URL: url})
	if res == true || err != nil {
		t.Errorf("Is recursive failed test %s", err)
	}
}

func TestKeyNameFromPrefix(t *testing.T) {
	var val string
	sm := &SecretsManagerResolver{}
	val = sm.keyNameFromPrefix("/hello/", "/hello/charles-dickens")
	if val != "CHARLES_DICKENS" {
		t.Errorf("Should match CHARLES_DICKENS, but got %s", val)
	}
}
