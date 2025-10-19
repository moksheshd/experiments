# Rust Practice

This repository is a **Cargo workspace** for experimenting with various Rust concepts and technologies. Each experiment is isolated in its own crate with independent dependencies and documentation.

## Repository Structure

This is a Cargo workspace containing multiple independent experiments:

```
rust-practice/
├── Cargo.toml              # Workspace configuration
├── README.md               # This file
│
├── erasure-coding/         # Erasure coding experiment
│   ├── Cargo.toml
│   ├── README.md           # Detailed experiment documentation
│   ├── src/
│   ├── examples/
│   └── tests/
│
└── [future-experiments]/   # More experiments will be added here
```

## Getting Started

### Prerequisites
- [Rust](https://www.rust-lang.org/tools/install) installed on your system (version 1.56+ for workspace inheritance)

### Working with Experiments

#### Run a specific experiment:
```bash
cargo run -p erasure-coding
```

#### Build a specific experiment:
```bash
cargo build -p erasure-coding
```

#### Test a specific experiment:
```bash
cargo test -p erasure-coding
```

#### Run an example from an experiment:
```bash
cargo run -p erasure-coding --example example_name
```

#### Build all experiments in the workspace:
```bash
cargo build --workspace
```

#### Test all experiments:
```bash
cargo test --workspace
```

### Working within an experiment directory

You can also `cd` into any experiment directory and use standard `cargo` commands:

```bash
cd erasure-coding
cargo run
cargo test
cargo build --release
```

## Current Experiments

### 1. Erasure Coding
**Location**: [erasure-coding/](erasure-coding/)

Exploration of erasure coding algorithms and their applications in distributed storage systems.

**Topics**: Reed-Solomon codes, data redundancy, fault tolerance, distributed systems

See [erasure-coding/README.md](erasure-coding/README.md) for detailed documentation.

---

## Adding New Experiments

To add a new experiment to the workspace:

1. Create a new directory for your experiment:
   ```bash
   mkdir my-experiment
   cd my-experiment
   cargo init
   ```

2. Add it to the workspace members in the root `Cargo.toml`:
   ```toml
   [workspace]
   members = [
       "erasure-coding",
       "my-experiment",  # Add your new experiment here
   ]
   ```

3. Update your experiment's `Cargo.toml` to inherit workspace settings:
   ```toml
   [package]
   name = "my-experiment"
   version.workspace = true
   edition.workspace = true
   ```

4. Create a detailed `README.md` in your experiment directory

## Practice Areas

This repository is designed for experimenting with:
- Advanced Rust concepts and algorithms
- Systems programming
- Distributed systems
- Performance optimization
- Data structures and algorithms
- Networking and protocols
- Cryptography and security
- And much more!

## Resources

- [The Rust Book](https://doc.rust-lang.org/book/)
- [Rust by Example](https://doc.rust-lang.org/rust-by-example/)
- [Cargo Workspaces Documentation](https://doc.rust-lang.org/book/ch14-03-cargo-workspaces.html)

---

**Happy coding!**
