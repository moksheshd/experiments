// Package config provides configuration functionality for the application
package config

import "time"

// Config holds application configuration
type Config struct {
	AppName        string
	Version        string
	LogLevel       string
	Timeout        time.Duration
	MaxConnections int
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		AppName:        "go-practice",
		Version:        "0.1.0",
		LogLevel:       "info",
		Timeout:        time.Second * 30,
		MaxConnections: 100,
	}
}

// GetConfig returns the application configuration
func GetConfig() Config {
	// In a real application, this would load from environment variables,
	// configuration files, etc.
	return DefaultConfig()
}
