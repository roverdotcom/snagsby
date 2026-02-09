package formatters

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
	actual := Merge(val)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Bad merge: %s != %s", actual, expected)
	}
}

func TestEnvFormat(t *testing.T) {
	in := map[string]string{
		"ONE": "1",
	}
	out := EnvFormater(in)
	expected := "export ONE=\"1\"\n"
	if strings.Compare(out, expected) != 0 {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("Env is off.")
	}

	in = map[string]string{
		"ESCAPE_TEST": `$HELLO "FRIEND" \12` + "`END",
	}
	out = EnvFormater(in)
	expected = `export ESCAPE_TEST="\$HELLO \"FRIEND\" \\12` + "\\`END\"\n"
	if strings.Compare(out, expected) != 0 {
		t.Error(out, expected)
	}
}

func TestJsonFormat(t *testing.T) {
	in := map[string]string{
		"B": "2",
		"A": "1",
		"Z": "10",
	}
	out := JSONFormater(in)
	expected := `{"A":"1","B":"2","Z":"10"}`
	if strings.Compare(out, expected) != 0 {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("Env single line is off.")
	}
}

func TestEnvFileFormat(t *testing.T) {
	in := map[string]string{
		"ONE": "1",
	}
	out := EnvFileFormater(in)
	expected := "ONE=\"1\"\n"
	if strings.Compare(out, expected) != 0 {
		fmt.Println(out)
		fmt.Println(expected)
		t.Errorf("EnvFile format is off.")
	}

	// Test with escaping
	in = map[string]string{
		"ESCAPE_TEST": `$HELLO "FRIEND" \12` + "`END",
	}
	out = EnvFileFormater(in)
	expected = `ESCAPE_TEST="\$HELLO \"FRIEND\" \\12` + "\\`END\"\n"
	if strings.Compare(out, expected) != 0 {
		t.Error(out, expected)
	}

	// Test with multiple keys (should be sorted)
	in = map[string]string{
		"Z": "last",
		"A": "first",
		"M": "middle",
	}
	out = EnvFileFormater(in)
	expected = "A=\"first\"\nM=\"middle\"\nZ=\"last\"\n"
	if strings.Compare(out, expected) != 0 {
		t.Errorf("EnvFile format sorting failed.\nGot:\n%s\nExpected:\n%s", out, expected)
	}
}
