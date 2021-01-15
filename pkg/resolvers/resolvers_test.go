package resolvers

import "testing"

func TestKeyRegexp(t *testing.T) {
	res := &Result{}
	res.AppendItem("Hello World/Test", "value")
	val, ok := res.Items["HELLO_WORLD_TEST"]
	if !ok || val != "value" {
		t.Errorf("Key not set correctly in %s", res.Items)
	}
}
