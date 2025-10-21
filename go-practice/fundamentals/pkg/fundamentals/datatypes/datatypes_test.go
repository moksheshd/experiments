package datatypes

import (
	"testing"
)

func TestDemoBasicTypes(t *testing.T) {
	// Ensures the demo function doesn't panic
	DemoBasicTypes()
}

func TestDemoNumericTypes(t *testing.T) {
	// Ensures the demo function doesn't panic
	DemoNumericTypes()
}

func TestDemoTypeConversions(t *testing.T) {
	// Ensures the demo function doesn't panic
	DemoTypeConversions()
}

func TestDemoCustomTypes(t *testing.T) {
	// Ensures the demo function doesn't panic
	DemoCustomTypes()
}

func TestSomeFunction(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        string
		wantBool bool
		wantInt  int
	}{
		{"basic test", 1, "test", true, -1},
		{"another test", 42, "hello", true, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBool, gotInt := SomeFunction(tt.a, tt.b)
			if gotBool != tt.wantBool {
				t.Errorf("SomeFunction() gotBool = %v, want %v", gotBool, tt.wantBool)
			}
			if gotInt != tt.wantInt {
				t.Errorf("SomeFunction() gotInt = %v, want %v", gotInt, tt.wantInt)
			}
		})
	}
}

func TestGetTypeInfo(t *testing.T) {
	info := GetTypeInfo()

	requiredTypes := []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "float32", "float64", "bool", "string"}

	for _, typeName := range requiredTypes {
		if _, exists := info[typeName]; !exists {
			t.Errorf("GetTypeInfo() missing type: %s", typeName)
		}
	}
}

func TestCustomType(t *testing.T) {
	ct := CustomType{
		Name:  "Test",
		Value: 123,
	}

	if ct.Name != "Test" {
		t.Errorf("CustomType.Name = %s, want Test", ct.Name)
	}
	if ct.Value != 123 {
		t.Errorf("CustomType.Value = %d, want 123", ct.Value)
	}
}

func TestTypeConversion(t *testing.T) {
	// Test int to float conversion
	i := 42
	f := float64(i)
	if f != 42.0 {
		t.Errorf("float64(42) = %f, want 42.0", f)
	}

	// Test float to int conversion (truncation)
	f2 := 42.9
	i2 := int(f2)
	if i2 != 42 {
		t.Errorf("int(42.9) = %d, want 42", i2)
	}

	// Test string to bytes and back
	str := "Hello"
	bytes := []byte(str)
	backToStr := string(bytes)
	if backToStr != str {
		t.Errorf("string conversion round-trip failed: got %s, want %s", backToStr, str)
	}
}

func ExampleSomeFunction() {
	o1, o2 := SomeFunction(10, "example")
	println(o1, o2)
	// Output varies, not checked
}

func ExampleDemoBasicTypes() {
	DemoBasicTypes()
	// Output is printed but not checked
}
