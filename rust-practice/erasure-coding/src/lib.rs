//! # Erasure Coding Library
//!
//! This library implements erasure coding experiments and demonstrations.
//!
//! ## Overview
//!
//! Erasure coding is a method of data protection in which data is broken into fragments,
//! expanded and encoded with redundant data pieces, and stored across different locations.
//!
//! ## Training Phases
//!
//! This library is structured as a progressive training program:
//!
//! ### Phase 1: XOR-Based Parity (Simple Redundancy)
//!
//! The simplest form of erasure coding using XOR operations. Learn the fundamentals
//! of parity-based data protection, similar to RAID-5.
//!
//! ```rust
//! use erasure_coding::phase1_xor_parity::*;
//!
//! let data = b"HELLO WORLD";
//! let encoded = encode(data, 3).unwrap();
//!
//! // Simulate losing chunk 1 and recover it
//! let recovered = recover_chunk(&encoded, 1).unwrap();
//! assert_eq!(recovered, encoded.data_chunks[1]);
//! ```
//!
//! ### Future Phases
//!
//! - **Phase 2**: Double Parity (RAID-6 style)
//! - **Phase 3**: Reed-Solomon Fundamentals
//! - **Phase 4**: Production Reed-Solomon
//! - **Phase 5**: Advanced Topics (Fountain Codes, etc.)
//!
//! ## Modules
//!
//! - [`phase1_xor_parity`]: XOR-based single parity protection

pub mod phase1_xor_parity;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_phase1_integration() {
        // Integration test to verify Phase 1 works correctly
        let data = b"Integration test data";
        let encoded = phase1_xor_parity::encode(data, 4).unwrap();

        // Verify we can recover any chunk
        for i in 0..4 {
            let recovered = phase1_xor_parity::recover_chunk(&encoded, i).unwrap();
            assert_eq!(recovered, encoded.data_chunks[i]);
        }
    }
}
