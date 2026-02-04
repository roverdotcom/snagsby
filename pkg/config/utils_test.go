package config

import (
	"os"
	"testing"
)

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"True mixed case", "True", true},
		{"1 as string", "1", true},
		{"yes lowercase", "yes", true},
		{"YES uppercase", "YES", true},
		{"Yes mixed case", "Yes", true},
		{"false", "false", false},
		{"0", "0", false},
		{"no", "no", false},
		{"empty string", "", false},
		{"random text", "random", false},
		{"2", "2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envName := "TEST_ENV_BOOL"
			os.Setenv(envName, tt.envValue)
			defer os.Unsetenv(envName)

			result := EnvBool(envName)
			if result != tt.expected {
				t.Errorf("EnvBool(%q=%q) = %v, expected %v", envName, tt.envValue, result, tt.expected)
			}
		})
	}

	// Test with unset environment variable
	t.Run("unset variable", func(t *testing.T) {
		result := EnvBool("NONEXISTENT_VAR")
		if result != false {
			t.Errorf("EnvBool(unset) = %v, expected false", result)
		}
	})
}
