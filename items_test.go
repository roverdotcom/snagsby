package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestQuoteEscaping(t *testing.T) {
	i := Item{Key: "test", Value: `hi "123"`}
	rendered := i.Export()
	expected := `export TEST="hi \"123\""`
	if rendered != expected {
		t.Errorf("%s != %s", expected, rendered)
	}
}

func TestCollection(t *testing.T) {
	c := NewCollection()
	c.AppendItem("test", "key")
	if c.Len() != 1 {
		t.Fail()
	}

	if c, _ := c.GetSecretString("TEST"); c != "key" {
		t.Fail()
	}

	if _, ok := c.GetSecretString("NO"); ok {
		t.Fail()
	}
}

func TestAppendItem(t *testing.T) {
	c := NewCollection()

	// We can write a wtring
	if e := c.AppendItem("hi", "world"); e != nil {
		t.Fail()
	}

	// Keys are upcased
	c.AppendItem("up", "case")
	if o, _ := c.GetSecretString("UP"); o != "case" {
		t.Fail()
	}

	// Spaces are not OK
	if e := c.AppendItem("a space", "value"); e == nil {
		t.Fail()
	}

	// Special characters are not OK
	if e := c.AppendItem("a[", "value"); e == nil {
		t.Fail()
	}

	// Numbers are OK
	if e := c.AppendItem("123isOK", "value"); e != nil {
		t.Fail()
	}
}

func TestReadSecretsParseFloats(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"number": 1.2
	}
	`))
	c.ReadSecretsFromReader(reader)
	if o, _ := c.GetSecretString("NUMBER"); o != "1.2" {
		t.Errorf("Looking for 1, got %s", o)
	}
}

// This currently panics, can we have it not?
func TestReadSecretsIncorrectJSON(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`["one", "two"]`))
	if err := c.ReadSecretsFromReader(reader); err == nil {
		t.Errorf("We should return an error on unparsable JSON")
	}
}

func TestReadSecretsParseBooleans(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"no": false,
		"yes": true
	}
	`))
	c.ReadSecretsFromReader(reader)
	if o, _ := c.GetSecretString("NO"); o != "0" {
		t.Errorf("Looking for 0, got %s", o)
	}
	if o, _ := c.GetSecretString("YES"); o != "1" {
		t.Errorf("Looking for 1, got %s", o)
	}
}

func TestReadSecretsParseStrings(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"hello": "world"
	}
	`))
	c.ReadSecretsFromReader(reader)
	if o, _ := c.GetSecretString("HELLO"); o != "world" {
		t.Errorf("Looking for HELLO, got %s", o)
	}
}

func TestReadSecretsIgnoresKeysWithSpaces(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"hello again": "world"
	}
	`))
	c.ReadSecretsFromReader(reader)
	if _, ok := c.GetSecretString("hello again"); ok {
		t.Errorf("Shouldn't have found hello again")
	}
}

func TestExportFormat(t *testing.T) {
	fmt.Println("This is a test")
	i := Item{Key: "hello", Value: "world"}
	expected := `export HELLO="world"`
	if i.Export() != expected {
		t.Errorf("Expected '%s' == '%s'", i.Export(), expected)
	}
}
