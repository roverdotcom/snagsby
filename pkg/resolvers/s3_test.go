package resolvers

import "testing"

func TestSanitizeKey(t *testing.T) {
	s := S3ManagerResolver{}
	values := [][]string{
		{"test/value", "test/value"},
		{"/test/value", "test/value"},
		{"//test/value", "/test/value"},
	}

	for _, value := range values {
		input := value[0]
		expected := value[1]
		actual := s.sanitizeKey(value[0])
		if actual != expected {
			t.Errorf("Input %s, expected %s, but got %s", input, expected, actual)
		}
	}
}
