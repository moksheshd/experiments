// Package datatypes demonstrates Go's type system and type conversions
package datatypes

import "fmt"

// DemoBasicTypes shows basic data type usage
func DemoBasicTypes() {
	fmt.Println("=== Data Types Example ===")

	var age int = 30
	fmt.Printf("Age is: %d\n", age)

	// Custom type based on int
	type IZ int
	var count IZ = 1
	fmt.Printf("Count: %d\n", count)

	var precision float32 = 10.1
	fmt.Println("Precision (float32):", precision)
	fmt.Println("Precision (converted to int):", int(precision))
}

// DemoNumericTypes shows different numeric types
func DemoNumericTypes() {
	fmt.Println("\n=== Numeric Types ===")

	// Integer types
	var i8 int8 = 127
	var i16 int16 = 32767
	var i32 int32 = 2147483647
	var i64 int64 = 9223372036854775807

	fmt.Printf("int8:  %d\n", i8)
	fmt.Printf("int16: %d\n", i16)
	fmt.Printf("int32: %d\n", i32)
	fmt.Printf("int64: %d\n", i64)

	// Unsigned integer types
	var ui8 uint8 = 255
	var ui16 uint16 = 65535

	fmt.Printf("uint8:  %d\n", ui8)
	fmt.Printf("uint16: %d\n", ui16)

	// Floating point types
	var f32 float32 = 3.14
	var f64 float64 = 3.141592653589793

	fmt.Printf("float32: %f\n", f32)
	fmt.Printf("float64: %.15f\n", f64)
}

// DemoTypeConversions shows how to convert between types
func DemoTypeConversions() {
	fmt.Println("\n=== Type Conversions ===")

	var i int = 42
	var f float64 = float64(i)
	var u uint = uint(f)

	fmt.Printf("int: %d -> float64: %f -> uint: %d\n", i, f, u)

	// String conversions
	var str string = "Hello"
	var bytes []byte = []byte(str)
	var backToStr string = string(bytes)

	fmt.Printf("string: %s -> []byte: %v -> string: %s\n", str, bytes, backToStr)
}

// SomeFunction demonstrates multiple return values with named returns
func SomeFunction(a int, b string) (o1 bool, o2 int) {
	// Named return values are initialized to zero values
	// We can explicitly return values
	return true, -1
}

// CustomType demonstrates creating custom types
type CustomType struct {
	Name  string
	Value int
}

// DemoCustomTypes shows custom type usage
func DemoCustomTypes() {
	fmt.Println("\n=== Custom Types ===")

	ct := CustomType{
		Name:  "Example",
		Value: 100,
	}

	fmt.Printf("CustomType: Name=%s, Value=%d\n", ct.Name, ct.Value)

	// Type alias
	type MyInt int
	var mi MyInt = 42
	fmt.Printf("MyInt (type alias): %d\n", mi)
}

// GetTypeInfo returns information about different types
func GetTypeInfo() map[string]string {
	return map[string]string{
		"int":     "Platform-dependent integer (32 or 64 bit)",
		"int8":    "8-bit signed integer (-128 to 127)",
		"int16":   "16-bit signed integer (-32768 to 32767)",
		"int32":   "32-bit signed integer",
		"int64":   "64-bit signed integer",
		"uint":    "Platform-dependent unsigned integer",
		"uint8":   "8-bit unsigned integer (0 to 255)",
		"float32": "32-bit floating point",
		"float64": "64-bit floating point",
		"bool":    "Boolean (true or false)",
		"string":  "String of characters",
	}
}
