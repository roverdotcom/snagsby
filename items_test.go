package main

import (
	"bytes"
	"testing"
)

func TestCollection(t *testing.T) {
	c := NewCollection()
	c.AppendItem("test", "key")
	if c.Len() != 1 {
		t.Fail()
	}

	if c, _ := c.GetItemString("TEST"); c != "key" {
		t.Fail()
	}

	if _, ok := c.GetItemString("NO"); ok {
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
	if o, _ := c.GetItemString("UP"); o != "case" {
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

func TestReadItemsParseFloats(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"number": 1.2
	}
	`))
	c.ReadItemsFromReader(reader)
	if o, _ := c.GetItemString("NUMBER"); o != "1.2" {
		t.Errorf("Looking for 1, got %s", o)
	}
}

// This currently panics, can we have it not?
func TestReadItemsIncorrectJSON(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`["one", "two"]`))
	if err := c.ReadItemsFromReader(reader); err == nil {
		t.Errorf("We should return an error on unparsable JSON")
	}
}

func TestReadItemsParseBooleans(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"no": false,
		"yes": true
	}
	`))
	c.ReadItemsFromReader(reader)
	if o, _ := c.GetItemString("NO"); o != "0" {
		t.Errorf("Looking for 0, got %s", o)
	}
	if o, _ := c.GetItemString("YES"); o != "1" {
		t.Errorf("Looking for 1, got %s", o)
	}
}

func TestReadItemsParseStrings(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"hello": "world"
	}
	`))
	c.ReadItemsFromReader(reader)
	if o, _ := c.GetItemString("HELLO"); o != "world" {
		t.Errorf("Looking for HELLO, got %s", o)
	}
}

func TestReadItemsIgnoresKeysWithSpaces(t *testing.T) {
	c := NewCollection()
	reader := bytes.NewReader([]byte(`
	{
		"hello again": "world"
	}
	`))
	c.ReadItemsFromReader(reader)
	if _, ok := c.GetItemString("hello again"); ok {
		t.Errorf("Shouldn't have found hello again")
	}
}
