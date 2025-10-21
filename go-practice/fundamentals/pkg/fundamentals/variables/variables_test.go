package variables

import (
	"testing"
)

func TestZeroValues(t *testing.T) {
	zeros := ZeroValues()

	tests := []struct {
		name     string
		key      string
		expected interface{}
	}{
		{"int zero value", "int", 0},
		{"float64 zero value", "float64", 0.0},
		{"bool zero value", "bool", false},
		{"string zero value", "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeros[tt.key]; got != tt.expected {
				t.Errorf("ZeroValues()[%q] = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestDemoBasicVariables(t *testing.T) {
	// This test ensures the demo function doesn't panic
	DemoBasicVariables()
}

func TestDemoMultipleVariables(t *testing.T) {
	// This test ensures the demo function doesn't panic
	DemoMultipleVariables()
}

func TestPrintZeroValues(t *testing.T) {
	// This test ensures the demo function doesn't panic
	PrintZeroValues()
}

// Example functions that appear in documentation
func ExampleZeroValues() {
	zeros := ZeroValues()
	println(zeros["int"].(int))
	println(zeros["bool"].(bool))
	// Output is not checked as it varies
}

func ExampleDemoBasicVariables() {
	DemoBasicVariables()
	// Output is printed but not checked in this example
}
