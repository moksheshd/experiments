package main

import (
	"fmt"
	"log"

	"github.com/mokshesh/go-practice/internal/config"
	"github.com/mokshesh/go-practice/pkg/greeting"
)

func main() {
	// Get application configuration
	cfg := config.GetConfig()
	log.Printf("Starting %s v%s...", cfg.AppName, cfg.Version)

	// Use the greeting package
	message := greeting.Greet("Developer")
	fmt.Println(message)

	// Greet multiple users
	names := []string{"Alice", "Bob", "Charlie"}
	greetings := greeting.GreetAll(names)

	fmt.Println("\nGreetings for multiple users:")
	for _, msg := range greetings {
		fmt.Println("-", msg)
	}
}
