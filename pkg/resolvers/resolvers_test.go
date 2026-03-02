package resolvers

import (
	"net/url"
	"testing"

	"github.com/roverdotcom/snagsby/pkg/config"
)

func TestKeyRegexp(t *testing.T) {
	res := &Result{}
	res.AppendItem("Hello World/Test", "value")
	val, ok := res.Items["HELLO_WORLD_TEST"]
	if !ok || val != "value" {
		t.Errorf("Key not set correctly in %s", res.Items)
	}
}

func TestAppendItems(t *testing.T) {
	res := &Result{}
	items := map[string]string{
		"key1":            "value1",
		"key2":            "value2",
		"Key-With-Dashes": "value3",
	}
	res.AppendItems(items)

	if len(res.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(res.Items))
	}
	if res.Items["KEY1"] != "value1" {
		t.Errorf("Expected KEY1=value1, got %s", res.Items["KEY1"])
	}
	if res.Items["KEY2"] != "value2" {
		t.Errorf("Expected KEY2=value2, got %s", res.Items["KEY2"])
	}
	if res.Items["KEY_WITH_DASHES"] != "value3" {
		t.Errorf("Expected KEY_WITH_DASHES=value3, got %s", res.Items["KEY_WITH_DASHES"])
	}
}

func TestAppendError(t *testing.T) {
	res := &Result{}

	if res.HasErrors() {
		t.Error("New result should not have errors")
	}

	err1 := &testError{"error 1"}
	res.AppendError(err1)

	if !res.HasErrors() {
		t.Error("Result should have errors after AppendError")
	}
	if len(res.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(res.Errors))
	}

	err2 := &testError{"error 2"}
	res.AppendError(err2)

	if len(res.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(res.Errors))
	}
}

func TestHasErrors(t *testing.T) {
	res := &Result{}

	if res.HasErrors() {
		t.Error("Empty result should not have errors")
	}

	res.Errors = []error{&testError{"test"}}
	if !res.HasErrors() {
		t.Error("Result with errors should return true")
	}
}

func TestItemKeys(t *testing.T) {
	res := &Result{}
	res.AppendItem("key1", "value1")
	res.AppendItem("key2", "value2")
	res.AppendItem("key3", "value3")

	keys := res.ItemKeys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify all keys are present (order doesn't matter)
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	if !keyMap["KEY1"] || !keyMap["KEY2"] || !keyMap["KEY3"] {
		t.Errorf("Missing expected keys in %v", keys)
	}
}

func TestLenItems(t *testing.T) {
	res := &Result{}

	if res.LenItems() != 0 {
		t.Errorf("Empty result should have 0 items, got %d", res.LenItems())
	}

	res.AppendItem("key1", "value1")
	if res.LenItems() != 1 {
		t.Errorf("Expected 1 item, got %d", res.LenItems())
	}

	res.AppendItem("key2", "value2")
	res.AppendItem("key3", "value3")
	if res.LenItems() != 3 {
		t.Errorf("Expected 3 items, got %d", res.LenItems())
	}
}

func TestResolveSource(t *testing.T) {
	// Test with invalid scheme
	invalidURL, _ := url.Parse("invalid://test/path")
	source := &config.Source{URL: invalidURL}
	result := ResolveSource(source)

	if !result.HasErrors() {
		t.Error("Expected error for invalid scheme")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error in Errors slice")
	}

	// Test with each valid scheme (will error due to no AWS, but scheme routing works)
	schemes := []string{"s3", "sm", "manifest"}
	for _, scheme := range schemes {
		testURL, _ := url.Parse(scheme + "://test/path")
		testSource := &config.Source{URL: testURL}
		result := ResolveSource(testSource)

		// Result should exist (even if it has errors due to missing AWS resources)
		if result == nil {
			t.Errorf("Expected result for scheme %s, got nil", scheme)
		}
		if result.Source != testSource {
			t.Errorf("Expected result.Source to match input source for scheme %s", scheme)
		}
	}
}

// Helper error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
