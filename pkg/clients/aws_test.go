package clients

import (
	"testing"
)

func TestReadJSONString(t *testing.T) {
	jsonStr := `{"hello": "world", "test": 12, "bool": false}`
	json, err := ReadJSONString(jsonStr)
	if err != nil ||
		json["HELLO"] != "world" ||
		json["TEST"] != "12" ||
		json["BOOL"] != "0" {
		t.Errorf("Failed to parse %s to %s", jsonStr, json)
	}
}
