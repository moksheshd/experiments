// Package greeting provides functions for creating greeting messages
package greeting

import "fmt"

// Greet returns a greeting message for the given name
func Greet(name string) string {
	return fmt.Sprintf("Hello, %s! Welcome to Go programming.", name)
}

// GreetAll returns a greeting message for multiple names
func GreetAll(names []string) []string {
	var greetings []string
	for _, name := range names {
		greetings = append(greetings, Greet(name))
	}
	return greetings
}
