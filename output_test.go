package main

import (
	"fmt"
	"reflect"
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
		"TWO": "2",
	}
	out := env(in)
	expected := `export ONE="1"
export TWO="2"
`
	if out != expected {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("Env is off.")
	}
}
