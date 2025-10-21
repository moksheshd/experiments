# Go Fundamentals

A comprehensive module covering the fundamental concepts of the Go programming language.

## Overview

This module provides hands-on examples, tests, and interactive demonstrations of core Go concepts. Each topic is self-contained with examples and tests to reinforce learning.

## Topics Covered

### 1. Variables
- Variable declarations
- Zero values
- Short variable declarations
- Multiple variable declarations

### 2. Data Types
- Basic types (int, float, bool, string)
- Type declarations
- Type conversions
- Custom types

### 3. Constants
- Constant declarations
- Grouped constants
- iota for enumerated constants
- Typed vs untyped constants

### 4. Functions
- Function declarations
- Multiple return values
- Named return values
- Variadic functions
- Closures

### 5. Control Flow
- If/else statements
- For loops
- Switch statements
- Defer, panic, and recover

### 6. Greeting Package
- Package organization
- Exported vs unexported identifiers
- Writing testable code

## Getting Started

### Run from workspace root:

```bash
# Run all examples
go run ./fundamentals/cmd/fundamentals

# Run tests
go test ./fundamentals/...

# Run tests with coverage
go test -cover ./fundamentals/...

# Run specific package tests
go test ./fundamentals/pkg/fundamentals/variables
go test ./fundamentals/pkg/fundamentals/datatypes
```

### Run from this directory:

```bash
cd fundamentals

# Run examples
go run ./cmd/fundamentals

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...
```

## Project Structure

```
fundamentals/
├── README.md                    # This file
├── go.mod                       # Module definition
├── pkg/
│   └── fundamentals/
│       ├── variables/           # Variable examples and tests
│       ├── datatypes/           # Data type examples and tests
│       ├── constants/           # Constants examples and tests
│       ├── functions/           # Function examples and tests
│       ├── controlflow/         # Control flow examples
│       └── greeting/            # Package example
├── cmd/
│   └── fundamentals/
│       └── main.go             # Interactive demo runner
└── examples/
    ├── variables_demo/         # Standalone variable examples
    ├── types_demo/             # Standalone type examples
    └── functions_demo/         # Standalone function examples
```

## Learning Path

1. **Start with Variables** - Understanding how to declare and use variables
2. **Move to Data Types** - Learn about Go's type system
3. **Explore Constants** - Master constant declarations and iota
4. **Study Functions** - Understand function declarations and closures
5. **Practice Control Flow** - Work with loops, conditionals, and switches
6. **Review Greeting Package** - See how to organize reusable code

## Self-Assessment Questions

After completing this module, you should be able to answer:

- What are zero values in Go and why do they matter?
- When should you use `:=` vs `var`?
- What is the difference between typed and untyped constants?
- How does `iota` work in constant declarations?
- What are the benefits of multiple return values?
- How do named return values affect function behavior?
- When should you use `defer`?
- What is the difference between exported and unexported identifiers?

## Resources

- [A Tour of Go](https://tour.golang.org/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Go Documentation](https://golang.org/doc/)

## Next Steps

After completing fundamentals, move on to:
- **Erasure Coding** - Apply Go knowledge to implement erasure coding algorithms
- **Concurrency** - Learn goroutines and channels
- **Data Structures** - Implement classic data structures in Go
- **Web Services** - Build HTTP services and REST APIs

---

**Happy Learning!**
