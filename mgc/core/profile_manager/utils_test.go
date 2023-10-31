package profile_manager

import (
	"testing"
)

func TestSanitizePath(t *testing.T) {
	type test struct {
		name     string
		input    string
		expected string
	}

	tests := []test{
		{
			name:     "",
			input:    "///bla../file",
			expected: "bla../file",
		},
		{
			name:     "",
			input:    "/////////file",
			expected: "file",
		},
		{
			name:     "",
			input:    "/absolute/path",
			expected: "absolute/path",
		},
		{
			name:     "",
			input:    "../../../../file",
			expected: "file",
		},
		{
			name:     "",
			input:    "/absolute/../../../relative/file",
			expected: "absolute/relative/file",
		},
		{
			name:     "",
			input:    "../",
			expected: "",
		},
		{
			name:     "",
			input:    "/",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := sanitizePath(tc.input)
			if output != tc.expected {
				t.Errorf("expected %s, found: %s", tc.expected, output)
			}
		})
	}
}
