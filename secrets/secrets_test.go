package secrets

import (
	"bytes"
	"testing"
)

func TestQuoteEscaping(t *testing.T) {
	s := Secret{Key: "test", Value: `hi "123"`}
	rendered := s.Export()
	expected := `export TEST="hi \"123\""`
	if rendered != expected {
		t.Errorf("%s != %s", expected, rendered)
	}
}

func TestCollection(t *testing.T) {
	s := NewCollection()
	s.WriteSecret("test", "key")
	if s.Len() != 1 {
		t.Fail()
	}

	if s, _ := s.GetSecretString("TEST"); s != "key" {
		t.Fail()
	}

	if _, ok := s.GetSecretString("NO"); ok {
		t.Fail()
	}
}

func TestWriteSecret(t *testing.T) {
	s := NewCollection()

	// We can write a wtring
	if e := s.WriteSecret("hi", "world"); e != nil {
		t.Fail()
	}

	// Keys are upcased
	s.WriteSecret("up", "case")
	if o, _ := s.GetSecretString("UP"); o != "case" {
		t.Fail()
	}

	// Spaces are not OK
	if e := s.WriteSecret("a space", "value"); e == nil {
		t.Fail()
	}

	// Special characters are not OK
	if e := s.WriteSecret("a[", "value"); e == nil {
		t.Fail()
	}

	// Numbers are OK
	if e := s.WriteSecret("123isOK", "value"); e != nil {
		t.Fail()
	}
}

func TestReadSecretsParseFloats(t *testing.T) {
	s := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"number": 1.2
	}
	`))
	s.ReadSecretsFromReader(reader)
	if o, _ := s.GetSecretString("NUMBER"); o != "1.2" {
		t.Errorf("Looking for 1, got %s", o)
	}
}

// This currently panics, can we have it not?
func TestReadSecretsIncorrectJSON(t *testing.T) {
	s := NewCollection()
	reader := bytes.NewReader([]byte(`["one", "two"]`))
	if err := s.ReadSecretsFromReader(reader); err == nil {
		t.Errorf("We should return an error on unparsable JSON")
	}
}

func TestReadSecretsParseBooleans(t *testing.T) {
	s := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"no": false,
		"yes": true
	}
	`))
	s.ReadSecretsFromReader(reader)
	if o, _ := s.GetSecretString("NO"); o != "0" {
		t.Errorf("Looking for 0, got %s", o)
	}
	if o, _ := s.GetSecretString("YES"); o != "1" {
		t.Errorf("Looking for 1, got %s", o)
	}
}

func TestReadSecretsParseStrings(t *testing.T) {
	s := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"hello": "world"
	}
	`))
	s.ReadSecretsFromReader(reader)
	if o, _ := s.GetSecretString("HELLO"); o != "world" {
		t.Errorf("Looking for HELLO, got %s", o)
	}
}

func TestReadSecretsIgnoresKeysWithSpaces(t *testing.T) {
	s := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"hello again": "world"
	}
	`))
	s.ReadSecretsFromReader(reader)
	if _, ok := s.GetSecretString("hello again"); ok {
		t.Errorf("Shouldn't have found hello again")
	}
}

func TestExportFormat(t *testing.T) {
	s := Secret{Key: "hello", Value: "world"}
	expected := `export HELLO="world"`
	if s.Export() != expected {
		t.Errorf("Expected '%s' == '%s'", s.Export(), expected)
	}
}
