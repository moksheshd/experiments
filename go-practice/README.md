# Go Practice

A **Go workspace** for experimenting with various Go concepts and technologies. Each learning goal is isolated in its own module with independent dependencies and comprehensive documentation.

## Repository Structure

This is a Go workspace containing multiple independent learning modules:

```
go-practice/
├── go.work                 # Workspace configuration
├── README.md               # This file
│
├── fundamentals/           # Learning Goal 1: Go Fundamentals
│   ├── go.mod
│   ├── README.md           # Detailed learning objectives
│   ├── pkg/
│   ├── cmd/
│   └── examples/
│
├── erasure-coding/         # Learning Goal 2: Erasure Coding Algorithms
│   ├── go.mod
│   ├── README.md           # Complete training curriculum
│   ├── pkg/
│   ├── cmd/
│   ├── examples/
│   └── benchmarks/
│
└── [future-modules]/       # More learning goals will be added here
```

## Getting Started

### Prerequisites
- [Go](https://golang.org/dl/) 1.21 or higher installed on your system

### Working with Modules

#### Run a specific module:
```bash
# Run fundamentals examples
go run ./fundamentals/cmd/fundamentals

# Run erasure-coding Phase 1 demo
go run ./erasure-coding/examples/phase1_xor_demo
```

#### Test a specific module:
```bash
# Test fundamentals
go test ./fundamentals/...

# Test erasure-coding
go test ./erasure-coding/...

# Test with coverage
go test -cover ./fundamentals/...
go test -cover ./erasure-coding/...
```

#### Run benchmarks:
```bash
go test -bench=. ./erasure-coding/pkg/erasurecoding/phase1
```

#### Test all modules in the workspace:
```bash
go test ./...
```

### Working within a module directory

You can also `cd` into any module directory and use standard `go` commands:

```bash
cd fundamentals
go test ./...
go test -v ./pkg/fundamentals/variables
go run ./cmd/fundamentals

cd ../erasure-coding
go test ./...
go run ./examples/phase1_xor_demo
```

## Current Learning Goals

### 1. Fundamentals
**Location**: [fundamentals/](fundamentals/)

**Status**: ✅ Core topics implemented

Master the fundamental concepts of the Go programming language through hands-on examples and comprehensive tests.

**Topics Covered:**
- Variables (declarations, zero values, short declarations)
- Data Types (basic types, custom types, type conversions)
- Constants (iota, enumerations, typed vs untyped)
- Functions (multiple returns, named returns, closures)
- Control Flow (loops, conditionals, switches)
- Package Organization (exported/unexported identifiers)

**Getting Started:**
```bash
# Run interactive fundamentals demo
go run ./fundamentals/cmd/fundamentals

# Run all tests
go test ./fundamentals/...
```

See [fundamentals/README.md](fundamentals/README.md) for detailed documentation.

---

### 2. Erasure Coding
**Location**: [erasure-coding/](erasure-coding/)

**Status**: ✅ Phase 1 complete, Phases 2-5 planned

Exploration of erasure coding algorithms and their applications in distributed storage systems. Progressive hands-on training from XOR parity to Reed-Solomon codes and fountain codes.

**Topics Covered:**
- **Phase 1** (✅ Complete): XOR-based parity (RAID-5 style)
- **Phase 2** (Planned): Double parity (RAID-6 style, Galois Fields)
- **Phase 3** (Planned): Reed-Solomon fundamentals
- **Phase 4** (Planned): Optimized Reed-Solomon with lookup tables
- **Phase 5** (Planned): Fountain codes and advanced topics

**Quick Start:**
```bash
# Run Phase 1 interactive demo
go run ./erasure-coding/examples/phase1_xor_demo

# Run tests
go test ./erasure-coding/...

# Run benchmarks
go test -bench=. ./erasure-coding/pkg/erasurecoding/phase1
```

**Why Erasure Coding?**
- Used in distributed storage (Ceph, MinIO, S3)
- Foundation of RAID systems
- Critical for data redundancy and fault tolerance
- Understand mathematical foundations (Galois Fields, Reed-Solomon)

See [erasure-coding/README.md](erasure-coding/README.md) for complete training curriculum.

---

### 3. Future Learning Goals

The following modules are planned for future implementation:

#### Concurrency (Planned)
- Goroutines and channels
- Common concurrency patterns
- Worker pools, pipelines, fan-out/fan-in
- Context and cancellation
- Race condition detection

#### Data Structures & Algorithms (Planned)
- Linked lists, trees, graphs
- Hash tables and maps
- Heaps and priority queues
- Sorting algorithms
- Algorithm analysis and benchmarking

#### Web Services (Planned)
- HTTP servers and clients
- REST API design
- Middleware patterns
- Request handling and routing
- Testing HTTP services

#### Database Integration (Planned)
- SQL database operations
- Connection pooling
- Transaction management
- ORM patterns
- Migration strategies

## Adding New Learning Goals

To add a new learning module to the workspace:

1. Create a new directory for your module:
   ```bash
   mkdir my-learning-goal
   cd my-learning-goal
   go mod init github.com/mokshesh/go-practice/my-learning-goal
   ```

2. Add it to the workspace in `go.work`:
   ```
   go 1.21

   use (
       ./fundamentals
       ./erasure-coding
       ./my-learning-goal     # Add your new module here
   )
   ```

3. Create a detailed `README.md` in your module directory with:
   - Learning objectives
   - Topics covered
   - Getting started guide
   - Self-assessment questions
   - Resources and references

4. Structure your module:
   ```
   my-learning-goal/
   ├── README.md           # Documentation
   ├── go.mod              # Module definition
   ├── pkg/                # Reusable packages
   ├── cmd/                # Command-line tools
   ├── examples/           # Standalone examples
   └── benchmarks/         # Performance tests (optional)
   ```

## Practice Philosophy

This repository is designed for:

- **Hands-on Learning**: Every concept has working code examples
- **Progressive Difficulty**: Start simple, build to advanced topics
- **Test-Driven**: Comprehensive tests demonstrate correct behavior
- **Benchmarked**: Performance considerations where relevant
- **Well-Documented**: Clear explanations and self-assessment questions
- **Production-Ready Patterns**: Learn both fundamentals and best practices

## Go Workspace Features

This project uses Go workspaces (introduced in Go 1.18) which allows:

- ✅ Multiple modules in a single repository
- ✅ Local development across modules without replace directives
- ✅ Shared dependencies managed at workspace level
- ✅ Independent versioning per module
- ✅ Clean separation of concerns

## Learning Path Recommendation

1. **Start with Fundamentals** (1-2 weeks)
   - Master Go syntax and basic concepts
   - Complete all topic exercises
   - Write and run tests

2. **Dive into Erasure Coding** (2-4 weeks)
   - Complete Phase 1 (XOR parity)
   - Understand the mathematical concepts
   - Implement Phases 2-5 progressively

3. **Explore Concurrency** (2-3 weeks, when implemented)
   - Master goroutines and channels
   - Learn concurrency patterns
   - Build real-world concurrent programs

4. **Build with Data Structures** (2-3 weeks, when implemented)
   - Implement classic algorithms
   - Analyze performance
   - Compare with standard library

5. **Create Web Services** (3-4 weeks, when implemented)
   - Build HTTP APIs
   - Implement middleware
   - Test thoroughly

## Testing Philosophy

All modules follow Go testing best practices:

```bash
# Run tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./fundamentals/pkg/fundamentals/variables

# Run benchmarks
go test -bench=. ./erasure-coding/...

# Run with race detector
go test -race ./...
```

## Code Quality

This repository demonstrates:

- **Idiomatic Go**: Following Go conventions and style guide
- **Error Handling**: Proper error types and handling patterns
- **Documentation**: godoc-compatible comments
- **Testing**: Table-driven tests, examples, and benchmarks
- **Package Design**: Clean interfaces and separation of concerns

## Resources

### Official Go Resources
- [A Tour of Go](https://tour.golang.org/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Go Documentation](https://golang.org/doc/)
- [Go Blog](https://blog.golang.org/)

### Testing & Benchmarking
- [Go Testing Package](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Benchmarking](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

### Module-Specific Resources
See individual module READMEs for curated learning resources.

## Progress Tracking

- ✅ **Fundamentals**: Core topics implemented
- ✅ **Erasure Coding**: Phase 1 complete, Phases 2-5 planned
- ⏳ **Concurrency**: Planned
- ⏳ **Data Structures**: Planned
- ⏳ **Web Services**: Planned

## Contributing

This is a personal learning repository, but contributions are welcome:

- Implement planned phases/modules
- Add more tests and examples
- Improve documentation
- Fix bugs or typos
- Suggest new learning goals

## License

This project is licensed under the MIT License - see individual module licenses for details.

---

**Repository Owner**: Mokshesh Dafariya (mokshesh@outlook.com)

**Last Updated**: 2025-10-22

**Happy Learning!**
