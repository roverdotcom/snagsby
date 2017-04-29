package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMerge(t *testing.T) {
	val := []map[string]string{
		map[string]string{
			"one":  "in one",
			"over": "from one",
		},
		map[string]string{
			"two":  "in two",
			"over": "from two",
			"new":  "new in two",
		},
		map[string]string{
			"three": "in three",
			"new":   "new in three",
		},
	}
	expected := map[string]string{
		"one":   "in one",
		"two":   "in two",
		"three": "in three",
		"over":  "from two",
		"new":   "new in three",
	}
	actual := merge(val)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Bad merge: %s != %s", actual, expected)
	}
}

func TestEnv(t *testing.T) {
	in := map[string]string{
		"ONE": "1",
	}
	out := env(in)
	expected := "export ONE=\"1\"\n"
	if strings.Compare(out, expected) != 0 {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("Env is off.")
	}
}
