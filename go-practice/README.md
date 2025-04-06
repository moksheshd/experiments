# Go Practice

A simple Go project demonstrating standard Go project structure and practices.

## Project Structure

```
go-practice/
├── cmd/            # Command-line applications
│   └── app/        # Main application
├── internal/       # Private application code
│   └── config/     # Application configuration
├── pkg/            # Reusable packages
│   └── greeting/   # Greeting functionality
├── .gitignore      # Git ignore file
├── go.mod          # Go module file
├── main.go         # Root level main.go for quick testing
└── README.md       # Project documentation
```

## Getting Started

### Prerequisites

- Go 1.16 or higher

### Building the Application

```bash
# Build the main application
go build -o bin/app ./cmd/app

# Run the application
./bin/app
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## Development

### Adding Dependencies

```bash
# Add a new dependency
go get github.com/example/package

# Update dependencies
go get -u ./...
```

### Code Style

This project follows the standard Go code style guidelines. You can use the following tools to ensure your code adheres to these guidelines:

```bash
# Format code
go fmt ./...

# Check for common mistakes
go vet ./...

# Run linter (requires golint)
golint ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
