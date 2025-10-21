// Package variables demonstrates variable declarations and usage in Go
package variables

import "fmt"

// DemoBasicVariables shows different ways to declare and initialize variables
func DemoBasicVariables() {
	fmt.Println("=== Variables Example ===")

	// Declare without initialization (zero value)
	var num int
	fmt.Printf("Uninitialized int: %d\n", num)

	var decision bool
	fmt.Printf("Uninitialized bool: %t\n", decision)

	// Declare with initialization
	var age int = 23
	fmt.Printf("Initialized int: %d\n", age)

	var isDeleted bool = true
	fmt.Printf("Initialized bool: %t\n", isDeleted)

	// Short variable declaration
	count := 4
	fmt.Printf("Short declaration: %d\n", count)
}

// DemoMultipleVariables shows multiple variable declarations
func DemoMultipleVariables() {
	fmt.Println("\n=== Multiple Variables ===")

	// Multiple variables in one line
	var x, y int = 1, 2
	fmt.Printf("x=%d, y=%d\n", x, y)

	// Multiple variables with short declaration
	name, age := "Alice", 30
	fmt.Printf("name=%s, age=%d\n", name, age)

	// Grouped variable declaration
	var (
		firstName string = "John"
		lastName  string = "Doe"
		userAge   int    = 25
	)
	fmt.Printf("User: %s %s, Age: %d\n", firstName, lastName, userAge)
}

// ZeroValues demonstrates zero values for different types
func ZeroValues() map[string]interface{} {
	var (
		i int
		f float64
		b bool
		s string
	)

	return map[string]interface{}{
		"int":     i,
		"float64": f,
		"bool":    b,
		"string":  s,
	}
}

// PrintZeroValues prints zero values for demonstration
func PrintZeroValues() {
	fmt.Println("\n=== Zero Values ===")
	zeros := ZeroValues()
	fmt.Printf("int:     %v\n", zeros["int"])
	fmt.Printf("float64: %v\n", zeros["float64"])
	fmt.Printf("bool:    %v\n", zeros["bool"])
	fmt.Printf("string:  %q\n", zeros["string"])
}
