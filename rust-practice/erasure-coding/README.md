# Erasure Coding Experiment

## Overview

This experiment explores erasure coding - a method of data protection where data is broken into fragments, expanded with redundant pieces, and stored across different locations or storage media.

## What is Erasure Coding?

Erasure coding is a data protection method that divides data into fragments, expands and encodes them with redundant data pieces, and stores them across different locations. Unlike simple replication, erasure coding provides the same level of data protection with significantly less storage overhead.

### Key Concepts

- **Data Shards**: Original data fragments (k shards)
- **Parity Shards**: Redundant encoded fragments (m shards)
- **Recovery**: Ability to reconstruct original data from any k out of (k + m) total shards
- **Storage Efficiency**: Uses less space than full replication while maintaining fault tolerance

### Common Algorithms

- **Reed-Solomon**: Most widely used, basis for QR codes, CDs/DVDs, RAID systems
- **Cauchy Reed-Solomon**: Optimized variant using Cauchy matrices
- **Fountain Codes**: Rateless codes that can generate infinite encoded symbols
- **LDPC**: Low-Density Parity-Check codes

## Use Cases

- Distributed storage systems (Ceph, MinIO, Cassandra)
- Cloud storage (Google, Amazon S3)
- RAID systems
- Network error correction
- Blockchain/cryptocurrency data availability

## Project Structure

```
erasure-coding/
├── src/
│   ├── lib.rs          # Core library implementation
│   ├── main.rs         # CLI interface and demos
│   └── [modules/]      # Additional modules as needed
├── examples/           # Usage examples and demonstrations
├── tests/              # Integration tests
└── README.md           # This file
```

## Getting Started

### Running the Experiment

From the workspace root:
```bash
# Run the main binary
cargo run -p erasure-coding

# Run tests
cargo test -p erasure-coding

# Run with release optimizations
cargo run -p erasure-coding --release
```

From this directory:
```bash
# Run the main binary
cargo run

# Run tests
cargo test

# Build only
cargo build
```

### Running Examples

```bash
# From workspace root
cargo run -p erasure-coding --example example_name

# From this directory
cargo run --example example_name
```

## Implementation Plan

### Phase 1: Foundation
- [ ] Understand Reed-Solomon algorithm fundamentals
- [ ] Choose implementation approach (from scratch vs. using crate)
- [ ] Set up basic data structures

### Phase 2: Core Functionality
- [ ] Implement encoding (data → shards)
- [ ] Implement decoding (shards → data)
- [ ] Handle edge cases and errors

### Phase 3: Testing & Validation
- [ ] Unit tests for encoding/decoding
- [ ] Integration tests with various data sizes
- [ ] Simulate shard loss and recovery

### Phase 4: Performance & Optimization
- [ ] Benchmark encoding/decoding performance
- [ ] Optimize hot paths
- [ ] Compare with existing implementations

### Phase 5: Practical Applications
- [ ] File splitting and reconstruction demo
- [ ] Simulate distributed storage scenario
- [ ] CLI tool for encoding/decoding files

## Dependencies

Currently none. Will add as needed:
- `reed-solomon-erasure` - Production-ready Reed-Solomon implementation
- `reed-solomon-simd` - SIMD-optimized variant
- `criterion` - For benchmarking
- `clap` - For CLI argument parsing (if building CLI tool)

## Learning Resources

### Papers & Articles
- [Reed-Solomon Codes](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction)
- [Erasure Coding in Windows Azure Storage](https://www.microsoft.com/en-us/research/publication/erasure-coding-in-windows-azure-storage/)

### Implementations
- [reed-solomon-erasure crate](https://docs.rs/reed-solomon-erasure/)
- [Backblaze Blog on Reed-Solomon](https://www.backblaze.com/blog/reed-solomon/)

### Videos & Tutorials
- Look for Reed-Solomon algorithm explanations
- Erasure coding in distributed systems talks

## Notes & Observations

_Add your findings, insights, and learnings here as you experiment._

## Performance Metrics

_Record performance measurements here._

## Challenges & Solutions

_Document problems encountered and how you solved them._

## Next Steps

_What to explore next after this experiment._

---

**Status**: Initial Setup Complete
**Started**: 2025-10-19
**Last Updated**: 2025-10-19
