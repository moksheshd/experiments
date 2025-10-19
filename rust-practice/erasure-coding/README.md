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

## 🎯 Training Path: From Theory to Practice

This project is structured as a **progressive, hands-on training program** to build deep understanding of erasure coding from first principles to production-quality implementations.

### Learning Approach

For each phase, you will:

1. **Read & Understand** - Learn the theory with clear explanations and diagrams
2. **Implement Together** - Write code step by step with guidance
3. **Test & Visualize** - Create tests that show exactly what's happening
4. **Interactive Demo** - Build CLI demos to see encoding/decoding in action
5. **Challenge Exercises** - Practice with modifications and extensions

### Phase 1: Foundation - XOR-Based Parity (Simple Redundancy)

**Goal:** Understand the core concept with the simplest possible implementation

**What You'll Learn:**
- How data chunks combine to create redundancy
- XOR operations and their reversibility properties
- RAID-5 style single parity protection
- Recovery from single chunk failure

**Implementation Tasks:**
- Split data into N chunks
- Generate 1 parity chunk using XOR operations
- Simulate losing any 1 chunk
- Recover the lost chunk perfectly
- Visualize binary operations

**Why This Matters:** This is the simplest form of erasure coding. You'll see how redundancy enables recovery without the complexity of advanced mathematics.

**Time Estimate:** 1-2 hours

### Phase 2: Multiple Parity - Double Protection

**Goal:** Extend to multiple parity chunks (RAID-6 style)

**What You'll Learn:**
- Multiple parity strategies
- P-parity (XOR-based) and Q-parity (Galois Field multiplication)
- Recovery from any 2 chunk failures
- Introduction to finite field arithmetic

**Implementation Tasks:**
- Implement dual parity generation
- Create Galois Field multiplication tables for GF(2^8)
- Implement recovery algorithms for 2-failure scenarios
- Test all possible 2-chunk failure combinations

**Why This Matters:** You'll start working with Galois Field arithmetic, which is the mathematical foundation of all modern erasure codes.

**Time Estimate:** 2-3 hours

### Phase 3: Reed-Solomon Fundamentals

**Goal:** Understand the mathematical foundation of modern erasure codes

**What You'll Learn:**
- Galois Fields (GF(2^8)) in depth
- Polynomial representation of data
- Vandermonde matrices and their properties
- Reed-Solomon encoding as polynomial evaluation
- Matrix inversion for decoding

**Implementation Tasks:**
- Implement complete GF(2^8) arithmetic (add, multiply, divide, power)
- Build polynomial evaluation functions
- Create encoding matrix (Vandermonde matrix)
- Implement matrix-based encoding
- Build decoding with Gaussian elimination

**Why This Matters:** Reed-Solomon is the industry standard erasure code. It's used in QR codes, CDs, DVDs, distributed storage systems, and space communications. Understanding it from scratch is invaluable.

**Time Estimate:** 4-6 hours

### Phase 4: Practical Reed-Solomon Implementation

**Goal:** Build production-quality code with real-world considerations

**What You'll Learn:**
- Systematic vs non-systematic encoding
- Optimized matrix operations
- Lookup table generation for fast arithmetic
- Handling variable chunk sizes
- Memory-efficient implementations
- Error handling and edge cases

**Implementation Tasks:**
- Implement systematic encoding (data shards unchanged)
- Generate and use pre-computed lookup tables
- Optimize encoding/decoding with SIMD or parallel operations
- Handle non-uniform data sizes with padding
- Add comprehensive error handling
- Create benchmarks

**Why This Matters:** Learn the difference between educational code and production code. Understand the trade-offs between simplicity, performance, and flexibility.

**Time Estimate:** 6-8 hours

### Phase 5: Advanced Topics & Real-World Use Cases

**Goal:** Explore modern variations and practical applications

**What You'll Learn:**
- Fountain codes (LT codes, Raptor codes)
- Streaming vs block erasure codes
- Integration patterns (how Ceph, MinIO, S3 use erasure coding)
- Network coding concepts
- Trade-offs: storage efficiency vs CPU cost vs recovery time

**Implementation Tasks:**
- Implement a simple Fountain code (LT code)
- Build a file splitter/reconstructor tool
- Create distributed storage simulation
- Benchmark different algorithms
- Compare with production libraries

**Why This Matters:** See how erasure coding is used in real systems. Understand when to use different approaches and their trade-offs.

**Time Estimate:** 8-12 hours

### Your First 30 Minutes

**Quick Start - Phase 1 Preview:**

1. Run the basic XOR parity demo:
   ```bash
   cargo run --example phase1_xor_demo
   ```

2. See encoding in action:
   ```
   Original Data: "HELLO WORLD"

   Split into 3 chunks:
   Chunk 0: "HELL"  (binary: 01001000 01000101 01001100 01001100)
   Chunk 1: "O WO"  (binary: 01001111 00100000 01010111 01001111)
   Chunk 2: "RLD!"  (binary: 01010010 01001100 01000100 00100001)

   Parity (XOR of all chunks): ...

   Now simulating loss of Chunk 1...
   Recovering using: Chunk0 XOR Chunk2 XOR Parity
   Recovered: "O WO" ✓
   ```

3. Modify the demo to:
   - Use your own text
   - Change the number of chunks
   - Try different failure scenarios

**You'll have your first "aha!" moment when you see data recovery in action!**

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
│   ├── lib.rs                    # Core library entry point
│   ├── main.rs                   # CLI interface with phase selection
│   │
│   ├── phase1_xor_parity.rs      # Phase 1: Simple XOR-based parity
│   ├── phase2_double_parity.rs   # Phase 2: RAID-6 style double parity
│   ├── phase3_galois_field.rs    # Phase 3: Galois Field GF(2^8) implementation
│   ├── phase3_reed_solomon.rs    # Phase 3: Reed-Solomon from scratch
│   ├── phase4_optimized_rs.rs    # Phase 4: Production-quality Reed-Solomon
│   ├── phase5_fountain_codes.rs  # Phase 5: Fountain codes (LT/Raptor)
│   │
│   └── utils/
│       ├── mod.rs
│       ├── matrix.rs             # Matrix operations for encoding/decoding
│       ├── galois.rs             # Galois Field arithmetic utilities
│       └── visualization.rs      # Pretty printing and visualization helpers
│
├── examples/
│   ├── phase1_xor_demo.rs        # Interactive XOR parity demo
│   ├── phase2_dual_parity.rs     # Dual parity demonstration
│   ├── phase3_rs_basics.rs       # Basic Reed-Solomon demo
│   ├── phase4_file_encoder.rs    # File splitting/reconstruction
│   └── phase5_streaming.rs       # Fountain code streaming demo
│
├── tests/
│   ├── integration_tests.rs      # End-to-end testing
│   └── phase_tests/              # Per-phase test suites
│
├── benches/
│   └── encoding_benchmark.rs     # Performance benchmarks
│
└── README.md                     # This file - complete training guide
```

## Getting Started

### Running the Training Program

**From the workspace root:**

```bash
# Run the main CLI (choose phases interactively)
cargo run -p erasure-coding

# Run a specific phase example
cargo run -p erasure-coding --example phase1_xor_demo
cargo run -p erasure-coding --example phase2_dual_parity
cargo run -p erasure-coding --example phase3_rs_basics

# Run tests for a specific phase
cargo test -p erasure-coding phase1
cargo test -p erasure-coding phase2

# Run all tests
cargo test -p erasure-coding

# Run benchmarks (Phase 4+)
cargo bench -p erasure-coding
```

**From this directory:**

```bash
# Run the main binary
cargo run

# Run examples
cargo run --example phase1_xor_demo

# Run tests
cargo test

# Run benchmarks
cargo bench
```

### Interactive Learning Mode

The main binary provides an interactive menu to explore each phase:

```bash
cargo run

# You'll see:
# Erasure Coding Training Program
# ===============================
#
# Select a phase to explore:
# 1. Phase 1: XOR-Based Parity
# 2. Phase 2: Double Parity (RAID-6)
# 3. Phase 3: Reed-Solomon Fundamentals
# 4. Phase 4: Optimized Reed-Solomon
# 5. Phase 5: Advanced Topics
#
# Enter your choice (1-5):
```

### Recommended Learning Path

1. **Start with Phase 1 Example:**
   ```bash
   cargo run --example phase1_xor_demo
   ```
   Experiment with the interactive demo, modify the code, and understand XOR parity.

2. **Read the Phase 1 Source Code:**
   Open [src/phase1_xor_parity.rs](src/phase1_xor_parity.rs) and study the implementation.

3. **Run Phase 1 Tests:**
   ```bash
   cargo test phase1 -- --nocapture
   ```
   See how tests validate the implementation.

4. **Complete Phase 1 Challenges** (see Progress Tracking section)

5. **Move to Phase 2** once you're comfortable with Phase 1

6. **Repeat** for each subsequent phase

## Progress Tracking

### Phase 1: XOR-Based Parity ⏳

**Core Implementation:**
- [ ] Create `phase1_xor_parity.rs` module
- [ ] Implement data chunking function
- [ ] Implement XOR parity generation
- [ ] Implement single chunk recovery
- [ ] Add binary visualization helpers

**Testing:**
- [ ] Test with various data sizes
- [ ] Test recovery of each chunk position
- [ ] Test edge cases (empty data, single byte, etc.)
- [ ] Visualize binary operations in tests

**Examples & Demos:**
- [ ] Create `phase1_xor_demo.rs` example
- [ ] Add interactive CLI for custom input
- [ ] Demonstrate all failure scenarios

**Challenges:**
- [ ] Modify to use 5 data chunks instead of 3
- [ ] Implement recovery of multiple chunks (if possible? understand limits)
- [ ] Create a visual diagram generator for encoding process
- [ ] Write a blog post explaining XOR parity to a beginner

**Self-Assessment:**
- [ ] Can you explain why XOR is reversible?
- [ ] Can you calculate parity by hand for small examples?
- [ ] Do you understand why this only protects against 1 failure?

### Phase 2: Double Parity (RAID-6) ⏳

**Core Implementation:**
- [ ] Create `phase2_double_parity.rs` module
- [ ] Implement P-parity (XOR)
- [ ] Implement Q-parity (Galois Field multiplication)
- [ ] Create GF(2^8) multiplication tables
- [ ] Implement 2-failure recovery algorithms

**Testing:**
- [ ] Test all single-failure scenarios
- [ ] Test all two-failure combinations
- [ ] Validate against known RAID-6 implementations
- [ ] Test with various chunk sizes

**Examples & Demos:**
- [ ] Create `phase2_dual_parity.rs` example
- [ ] Visualize both parity calculations
- [ ] Interactive demo for 2-failure scenarios

**Challenges:**
- [ ] Derive the Q-parity formula yourself
- [ ] Implement recovery using matrix inversion
- [ ] Compare performance: XOR vs GF multiplication
- [ ] Explain when you'd use RAID-6 vs RAID-5

**Self-Assessment:**
- [ ] Can you explain what a Galois Field is?
- [ ] Do you understand why we can't just XOR twice?
- [ ] Can you perform GF(2^8) multiplication by hand?

### Phase 3: Reed-Solomon Fundamentals ⏳

**Core Implementation:**
- [ ] Create `phase3_galois_field.rs` module
- [ ] Implement complete GF(2^8) arithmetic (add, multiply, divide, power)
- [ ] Implement polynomial evaluation
- [ ] Create `phase3_reed_solomon.rs` module
- [ ] Build Vandermonde matrix generator
- [ ] Implement matrix-based encoding
- [ ] Implement Gaussian elimination for decoding
- [ ] Handle arbitrary (k, m) configurations

**Testing:**
- [ ] Test GF arithmetic against known values
- [ ] Validate polynomial evaluation
- [ ] Test encoding with various (k, m) values
- [ ] Test decoding with all failure combinations
- [ ] Verify systematic encoding property

**Examples & Demos:**
- [ ] Create `phase3_rs_basics.rs` example
- [ ] Visualize matrix operations
- [ ] Show polynomial evaluation process
- [ ] Interactive (k, m) configuration

**Challenges:**
- [ ] Prove why Vandermonde matrices are invertible
- [ ] Implement logarithm tables for fast multiplication
- [ ] Optimize matrix multiplication
- [ ] Compare systematic vs non-systematic encoding

**Self-Assessment:**
- [ ] Can you construct a Vandermonde matrix by hand?
- [ ] Do you understand the polynomial interpretation?
- [ ] Can you explain the decoding process to someone?
- [ ] Why is GF(2^8) chosen (vs GF(2^4) or GF(2^16))?

### Phase 4: Production Reed-Solomon ⏳

**Core Implementation:**
- [ ] Create `phase4_optimized_rs.rs` module
- [ ] Implement systematic encoding
- [ ] Generate pre-computed lookup tables
- [ ] Add SIMD optimizations (if applicable)
- [ ] Implement memory-efficient streaming
- [ ] Add comprehensive error handling
- [ ] Handle variable chunk sizes with padding

**Testing:**
- [ ] Benchmark against naive implementation
- [ ] Test with large datasets (MB, GB range)
- [ ] Validate correctness vs Phase 3
- [ ] Memory usage profiling
- [ ] Concurrent encoding/decoding tests

**Examples & Demos:**
- [ ] Create `phase4_file_encoder.rs` example
- [ ] Build file splitter/reconstructor CLI
- [ ] Simulate distributed storage

**Challenges:**
- [ ] Compare performance with `reed-solomon-erasure` crate
- [ ] Add multi-threaded encoding
- [ ] Implement incremental encoding
- [ ] Profile and optimize hot paths

**Self-Assessment:**
- [ ] What are the performance bottlenecks?
- [ ] When would you use lookup tables vs direct computation?
- [ ] Can you explain the systematic encoding advantage?

### Phase 5: Advanced Topics ⏳

**Core Implementation:**
- [ ] Create `phase5_fountain_codes.rs` module
- [ ] Implement LT (Luby Transform) codes
- [ ] Implement degree distribution
- [ ] Build streaming encoder/decoder
- [ ] Compare with Reed-Solomon

**Research & Analysis:**
- [ ] Study how Ceph uses erasure coding
- [ ] Analyze MinIO's implementation
- [ ] Research S3 Glacier's approach
- [ ] Understand Raptor codes

**Examples & Demos:**
- [ ] Create `phase5_streaming.rs` example
- [ ] Build network simulation
- [ ] Benchmark different algorithms

**Challenges:**
- [ ] When would you choose Fountain codes over RS?
- [ ] Implement Raptor codes
- [ ] Compare CPU cost vs storage efficiency
- [ ] Build a complete distributed storage prototype

**Self-Assessment:**
- [ ] Can you compare different erasure code families?
- [ ] Do you understand the trade-offs?
- [ ] Could you choose the right algorithm for a use case?

---

### Overall Progress Summary

- **Phase 1**: ⬜ Not Started | ⏳ In Progress | ✅ Complete
- **Phase 2**: ⬜ Not Started | ⏳ In Progress | ✅ Complete
- **Phase 3**: ⬜ Not Started | ⏳ In Progress | ✅ Complete
- **Phase 4**: ⬜ Not Started | ⏳ In Progress | ✅ Complete
- **Phase 5**: ⬜ Not Started | ⏳ In Progress | ✅ Complete

**Current Phase**: Phase 1 - XOR-Based Parity
**Started**: 2025-10-20
**Target Completion**: _Set your own pace!_

## Dependencies

### Phase 1-3 (Learning)
No external dependencies needed! Build everything from scratch to learn deeply.

### Phase 4 (Optimization)
- `criterion` - For detailed performance benchmarking
- `clap` - For building CLI tools

### Phase 5 (Comparison)
- `reed-solomon-erasure` - Compare with production implementation
- `reed-solomon-simd` - Study SIMD optimizations

Add dependencies as you progress through phases.

## Learning Resources

### Phase 1: XOR & Parity Fundamentals

**Beginner-Friendly:**
- [XOR - The Magic Bit Operation](https://www.khanacademy.org/computing/computer-science/cryptography/ciphers/a/xor-bitwise-operation) - Khan Academy
- [How RAID 5 Works](https://www.youtube.com/results?search_query=raid+5+explained) - Visual explanations (YouTube)
- [Parity Bits Explained](https://en.wikipedia.org/wiki/Parity_bit) - Wikipedia

**Hands-On:**
- Practice XOR operations on paper with small examples
- Use a binary calculator to verify your understanding

### Phase 2: RAID-6 & Galois Fields

**Conceptual:**
- [RAID 6 Explained](https://www.prepressure.com/library/technology/raid) - Visual guide
- [Introduction to Galois Fields](http://www.cs.utsa.edu/~wagner/laws/FFM.html) - Beginner-friendly intro

**Technical:**
- [Galois Field Arithmetic](https://en.wikipedia.org/wiki/Finite_field_arithmetic) - Wikipedia
- [RAID-6 Recovery Math](https://www.kernel.org/pub/linux/kernel/people/hpa/raid6.pdf) - H. Peter Anvin's paper

**Interactive:**
- [Galois Field Calculator](http://www.ee.unb.ca/cgi-bin/tervo/calc2.pl) - Online tool to verify calculations

### Phase 3: Reed-Solomon Theory

**Essential Reading:**
- [Reed-Solomon Error Correction](https://downloads.bbc.co.uk/rd/pubs/whp/whp-pdf-files/WHP031.pdf) - BBC R&D White Paper (excellent!)
- [Reed-Solomon Codes for Coders](https://en.wikiversity.org/wiki/Reed%E2%80%93Solomon_codes_for_coders) - Wikiversity tutorial
- [The Mathematics of Erasure Codes](https://faculty.math.illinois.edu/~reznick/390.pdf) - Academic but accessible

**Video Lectures:**
- [Reed-Solomon Coding Theory](https://www.youtube.com/results?search_query=reed+solomon+coding+theory) - Search YouTube
- [Error Correcting Codes](https://www.coursera.org/learn/coding-theory) - Coursera (free to audit)

**Papers:**
- [Reed-Solomon Codes](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) - Wikipedia (comprehensive)
- [A Tutorial on Reed-Solomon Coding](https://ntrs.nasa.gov/api/citations/19900019023/downloads/19900019023.pdf) - NASA Technical Report

### Phase 4: Production Implementation

**Engineering Resources:**
- [Backblaze Blog on Reed-Solomon](https://www.backblaze.com/blog/reed-solomon/) - Practical perspective
- [Optimizing Galois Field Arithmetic](https://www.ssrc.ucsc.edu/media/pubs/5495f7109e0fbbee67c89ae5daa0c9e0db83d5f1.pdf) - Research paper
- [SIMD Optimization Techniques](https://www.intel.com/content/www/us/en/docs/intrinsics-guide/index.html) - Intel Intrinsics Guide

**Implementations to Study:**
- [reed-solomon-erasure crate](https://docs.rs/reed-solomon-erasure/) - Rust implementation
- [Jerasure Library](https://github.com/tsuraan/Jerasure) - C implementation
- [Intel ISA-L](https://github.com/intel/isa-l) - Highly optimized library

### Phase 5: Advanced Topics & Applications

**Fountain Codes:**
- [LT Codes (Luby Transform)](https://en.wikipedia.org/wiki/Luby_transform_code) - Wikipedia
- [Digital Fountain Codes](https://web.cs.ucla.edu/~rafail/PUBLIC/116.pdf) - Original Luby paper
- [Raptor Codes](https://www.qualcomm.com/media/documents/files/raptor-codes.pdf) - Qualcomm whitepaper

**Distributed Storage Systems:**
- [Erasure Coding in Windows Azure Storage](https://www.microsoft.com/en-us/research/publication/erasure-coding-in-windows-azure-storage/) - Microsoft Research
- [Ceph Erasure Coding](https://docs.ceph.com/en/latest/rados/operations/erasure-code/) - Official docs
- [Facebook's f4: Warm BLOB Storage](https://www.usenix.org/system/files/conference/osdi14/osdi14-paper-muralidhar.pdf) - OSDI paper

**Blockchain & Web3:**
- [Polkadot's Availability and Validity](https://wiki.polkadot.network/docs/learn-availability) - Erasure coding in blockchain
- [Data Availability Sampling](https://arxiv.org/abs/1809.09044) - Celestia's approach

**Comparison & Analysis:**
- [Erasure Codes for Storage Systems](https://dl.acm.org/doi/10.1145/2602040) - ACM survey paper
- [Benchmark of Erasure Coding Libraries](https://github.com/Backblaze/Backblaze-Open-Source-Reed-Solomon) - Backblaze comparison

### Recommended Reading Order

**Week 1:** Phase 1 resources + XOR fundamentals
**Week 2:** Phase 2 resources + Galois Field basics
**Week 3-4:** Phase 3 resources + Reed-Solomon theory (take your time!)
**Week 5-6:** Phase 4 resources + optimization techniques
**Week 7+:** Phase 5 resources + explore your interests

### Interactive Tools

- [Galois Field Calculator](http://www.ee.unb.ca/cgi-bin/tervo/calc2.pl)
- [Matrix Calculator](https://www.wolframalpha.com/) - WolframAlpha for matrix operations
- [Binary/Hex Converter](https://www.rapidtables.com/convert/number/binary-to-hex.html)

### Communities & Forums

- [r/coding](https://www.reddit.com/r/coding/) - General CS questions
- [Stack Overflow](https://stackoverflow.com/questions/tagged/reed-solomon) - Reed-Solomon tag
- [Cryptography Stack Exchange](https://crypto.stackexchange.com/) - Error correction questions

## Interactive Examples & Expected Outputs

### Phase 1: XOR Parity Demo

```bash
$ cargo run --example phase1_xor_demo

Erasure Coding - Phase 1: XOR-Based Parity
==========================================

Enter your data (or press Enter for default "HELLO WORLD"):
> Test Data

Number of data chunks (2-10, default 3):
> 3

Encoding Process:
-----------------
Original: "Test Data" (9 bytes)

Chunk 0: "Tes" (3 bytes)
  Binary: 01010100 01100101 01110011

Chunk 1: "t D" (3 bytes)
  Binary: 01110100 00100000 01000100

Chunk 2: "ata" (3 bytes)
  Binary: 01100001 01110100 01100001

Parity Chunk (XOR of all chunks):
  Binary: 01000001 01000001 00110110
  ASCII:  "AA6"

Total storage: 4 chunks (original 3 + 1 parity)
Storage overhead: 33.3%

Recovery Demonstration:
-----------------------
Simulating loss of Chunk 1...

Recovery calculation:
  Chunk 0 XOR Chunk 2 XOR Parity
  = "Tes" XOR "ata" XOR "AA6"
  = 01010100 XOR 01100001 XOR 01000001 (first byte)
  = 01110100 (ASCII: 't')

Recovered Chunk 1: "t D" ✓ SUCCESS!

Original data fully reconstructed!
```

### Phase 2: Dual Parity Demo

```bash
$ cargo run --example phase2_dual_parity

Erasure Coding - Phase 2: Double Parity (RAID-6)
=================================================

Configuration: 4 data chunks + 2 parity chunks

Data Chunks:
  D0: "ABCD"
  D1: "EFGH"
  D2: "IJKL"
  D3: "MNOP"

P-Parity (XOR):
  D0 ⊕ D1 ⊕ D2 ⊕ D3 = ...

Q-Parity (Galois Field):
  (1*D0) ⊕ (g¹*D1) ⊕ (g²*D2) ⊕ (g³*D3) = ...
  where g is the generator in GF(2⁸)

Testing 2-Failure Scenarios:
-----------------------------
Scenario 1: Lost D0 and D1
  Using P and Q parity to solve system of equations...
  Recovered: D0="ABCD", D1="EFGH" ✓

Scenario 2: Lost D2 and P-Parity
  Using remaining chunks and Q-Parity...
  Recovered: D2="IJKL", P="..." ✓

All 15 possible 2-failure combinations tested: ✓ PASSED
```

### Phase 3: Reed-Solomon Basics

```bash
$ cargo run --example phase3_rs_basics

Erasure Coding - Phase 3: Reed-Solomon Fundamentals
====================================================

Configuration: k=4 data shards, m=2 parity shards

Step 1: Galois Field GF(2⁸) Setup
----------------------------------
Primitive polynomial: x⁸ + x⁴ + x³ + x² + 1
Generator element: 2

Sample GF operations:
  5 ⊕ 3 = 6 (XOR)
  5 ⊗ 3 = 15 (GF multiplication)
  5 / 3 = 6 (GF division)

Step 2: Encoding Matrix (Vandermonde)
--------------------------------------
[1  1  1  1]     [D0]   [D0]
[1  2  4  8]  ×  [D1] = [D1]
[1  3  9 27]     [D2]   [D2]
[1  4 16 64]     [D3]   [D3]
[1  5 25125]            [P0]
[1  6 36216]            [P1]

Step 3: Encoding Data
---------------------
Input: "ERASURE_CODING_TEST"
Data shards (4 bytes each):
  D0: "ERAS"
  D1: "URE_"
  D2: "CODI"
  D3: "NG_T"

Computing parity shards using matrix multiplication...
  P0: [calculated bytes]
  P1: [calculated bytes]

Storage: 6 shards total (4 data + 2 parity)

Step 4: Simulating Failures & Recovery
---------------------------------------
Lost: D1 and P0

Decoding matrix (using available shards):
[1  1  1  1]   Available: D0, D2, D3, P1
Using Gaussian elimination to invert...

Recovered D1: "URE_" ✓
Recovered P0: [...] ✓

Full data reconstructed!
```

### Phase 4: File Encoder Tool

```bash
$ cargo run --example phase4_file_encoder -- encode myfile.txt --shards 10 --parity 4

Erasure Coding - Phase 4: File Encoder
=======================================

Input file: myfile.txt (1,234,567 bytes)
Configuration: 10 data shards + 4 parity shards

Encoding...
  Chunk size: 123,457 bytes (with padding)
  Data shards: 10 × 123,457 = 1,234,570 bytes
  Parity shards: 4 × 123,457 = 493,828 bytes
  Total storage: 1,728,398 bytes

Storage overhead: 40% (can lose any 4 shards)
Encoding time: 45.2 ms
Throughput: 27.3 MB/s

Output shards written to:
  myfile.txt.shard.0 ... myfile.txt.shard.9 (data)
  myfile.txt.shard.10 ... myfile.txt.shard.13 (parity)

$ cargo run --example phase4_file_encoder -- decode myfile.txt.shard.* --output recovered.txt

Decoding...
  Found 14 shards (need minimum 10)
  Missing: none
  Decoding time: 38.1 ms

Recovered file: recovered.txt (1,234,567 bytes)
Verification: ✓ MATCHES ORIGINAL (SHA256 verified)
```

### Phase 5: Fountain Code Streaming

```bash
$ cargo run --example phase5_streaming

Erasure Coding - Phase 5: LT Fountain Codes
============================================

Input data: 1000 bytes
Encoding symbols: generating infinite stream...

Symbol   1: [encoded data] (degree 3)
Symbol   2: [encoded data] (degree 1)
Symbol   3: [encoded data] (degree 5)
...

Simulating lossy channel (30% packet loss)...
  Sent: 1500 symbols
  Received: 1050 symbols
  Decoding progress: ████████████ 100%

Data fully recovered after receiving 1072 symbols!
Overhead: 7.2% (theoretical minimum: ~5%)

Key advantage: No need to retransmit specific lost packets!
```

## Notes & Observations

### Things You'll Discover

**Phase 1 Insights:**
- XOR is its own inverse: `A ⊕ B ⊕ B = A`
- Order doesn't matter: `A ⊕ B ⊕ C = C ⊕ A ⊕ B`
- This is the foundation of ALL erasure codes!

**Phase 2 Insights:**
- Two different mathematical operations are needed for 2-failure recovery
- Galois Fields make division always possible (no division by zero!)
- RAID-6 uses this in production every day

**Phase 3 Insights:**
- Reed-Solomon views data as polynomial coefficients
- Matrix inversion is the key to decoding
- The Vandermonde matrix structure is special and always invertible

**Phase 4 Insights:**
- Lookup tables trade memory for speed
- Pre-computing matrices saves time during encoding
- Real-world considerations: padding, alignment, error handling

**Phase 5 Insights:**
- Fountain codes can generate infinite encoded symbols
- Different codes for different use cases (storage vs streaming)
- Trade-offs: CPU cost, storage overhead, recovery time

### Performance Metrics

_Record your measurements here as you progress:_

```
Phase 3 vs Phase 4 (Reed-Solomon):
  Naive implementation: ___ MB/s
  Optimized with lookup tables: ___ MB/s
  Speedup: ___x

Comparison with production libraries:
  Your implementation: ___ MB/s
  reed-solomon-erasure crate: ___ MB/s

Memory usage:
  Phase 3: ___ MB
  Phase 4: ___ MB
```

### Challenges & Solutions

_Document your journey:_

**Example:**
- **Challenge**: Gaussian elimination was numerically unstable
- **Solution**: Realized I was using regular arithmetic instead of GF arithmetic!
- **Lesson**: Every operation must be in the Galois Field

### Next Steps After Completion

Once you've completed all 5 phases, consider:

1. **Build a real application:**
   - Distributed backup tool
   - RAID implementation
   - Network protocol with FEC

2. **Explore related topics:**
   - LDPC codes (used in 5G, WiFi)
   - Turbo codes (used in satellite communications)
   - Network coding (beyond erasure coding)

3. **Contribute to open source:**
   - Optimize existing libraries
   - Add features to `reed-solomon-erasure`
   - Write tutorials for others

4. **Apply to your domain:**
   - How could erasure coding help your current projects?
   - Could it improve reliability or reduce costs?

---

**Status**: Ready to Begin Training
**Started**: 2025-10-20
**Last Updated**: 2025-10-20

**Current Phase**: Phase 1 - XOR-Based Parity
**Next Milestone**: Complete Phase 1 implementation and demos

---

## Getting Help

If you get stuck:

1. Review the Learning Resources for your current phase
2. Check the self-assessment questions - can you answer them?
3. Look at the example outputs - does yours match?
4. Read through the source code of completed phases
5. Search for your specific question on Stack Overflow
6. Remember: struggling is part of learning! Don't skip the hard parts.

**Ready to start? Run this command to begin Phase 1:**

```bash
# Let's build your first erasure code!
cargo run --example phase1_xor_demo
```

Good luck on your erasure coding journey!
