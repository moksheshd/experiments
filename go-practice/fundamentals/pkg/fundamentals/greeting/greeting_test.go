package greeting

import (
	"testing"
)

func TestGreet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular name",
			input:    "John",
			expected: "Hello, John! Welcome to Go programming.",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "Hello, ! Welcome to Go programming.",
		},
		{
			name:     "special characters",
			input:    "John & Jane",
			expected: "Hello, John & Jane! Welcome to Go programming.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Greet(tt.input)
			if result != tt.expected {
				t.Errorf("Greet(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGreetAll(t *testing.T) {
	names := []string{"Alice", "Bob", "Charlie"}
	results := GreetAll(names)

	if len(results) != len(names) {
		t.Errorf("GreetAll() returned %d results, want %d", len(results), len(names))
	}

	for i, name := range names {
		expected := Greet(name)
		if results[i] != expected {
			t.Errorf("GreetAll()[%d] = %q, want %q", i, results[i], expected)
		}
	}
}
