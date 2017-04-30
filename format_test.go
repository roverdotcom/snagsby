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

func TestEnvFormat(t *testing.T) {
	in := map[string]string{
		"ONE": "1",
	}
	out := EnvFormat(in)
	expected := "export ONE=\"1\"\n"
	if strings.Compare(out, expected) != 0 {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("Env is off.")
	}
}

func TestJsonFormat(t *testing.T) {
	in := map[string]string{
		"B": "2",
		"A": "1",
		"Z": "10",
	}
	out := JSONFormat(in)
	expected := `{"A":"1","B":"2","Z":"10"}`
	if strings.Compare(out, expected) != 0 {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("Env single line is off.")
	}
}
