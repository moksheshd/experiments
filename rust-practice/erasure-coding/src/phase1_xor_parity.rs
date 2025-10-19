//! # Phase 1: XOR-Based Parity (Simple Redundancy)
//!
//! This module implements the simplest form of erasure coding using XOR operations.
//! It demonstrates RAID-5 style single parity protection.
//!
//! ## Key Concepts
//!
//! - Split data into N chunks
//! - Generate 1 parity chunk using XOR of all data chunks
//! - Can recover from any single chunk failure
//! - XOR is reversible: A ⊕ B ⊕ B = A
//!
//! ## Example
//!
//! ```rust
//! use erasure_coding::phase1_xor_parity::*;
//!
//! let data = b"HELLO WORLD";
//! let num_chunks = 3;
//!
//! // Encode: split into chunks and generate parity
//! let encoded = encode(data, num_chunks).unwrap();
//! println!("Data chunks: {}", encoded.data_chunks.len());
//! println!("Parity chunk: {} bytes", encoded.parity_chunk.len());
//!
//! // Simulate losing chunk 1
//! let recovered = recover_chunk(&encoded, 1).unwrap();
//! assert_eq!(recovered, encoded.data_chunks[1]);
//! ```

use std::fmt;

/// Represents encoded data with XOR parity
#[derive(Debug, Clone, PartialEq)]
pub struct XorEncoded {
    /// Original data chunks
    pub data_chunks: Vec<Vec<u8>>,
    /// XOR parity chunk (same size as each data chunk)
    pub parity_chunk: Vec<u8>,
    /// Size of each chunk in bytes
    pub chunk_size: usize,
}

/// Errors that can occur during encoding or recovery
#[derive(Debug, PartialEq)]
pub enum XorError {
    /// Input data is empty
    EmptyData,
    /// Number of chunks must be at least 2
    InvalidChunkCount,
    /// Chunk index is out of bounds
    InvalidChunkIndex,
}

impl fmt::Display for XorError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            XorError::EmptyData => write!(f, "Input data cannot be empty"),
            XorError::InvalidChunkCount => write!(f, "Number of chunks must be at least 2"),
            XorError::InvalidChunkIndex => write!(f, "Chunk index is out of bounds"),
        }
    }
}

impl std::error::Error for XorError {}

/// Encodes data by splitting it into chunks and generating XOR parity
///
/// # Arguments
///
/// * `data` - The input data to encode
/// * `num_chunks` - Number of data chunks to split into (must be >= 2)
///
/// # Returns
///
/// An `XorEncoded` structure containing the data chunks and parity chunk
///
/// # Errors
///
/// Returns `XorError::EmptyData` if input is empty
/// Returns `XorError::InvalidChunkCount` if num_chunks < 2
///
/// # Example
///
/// ```rust
/// use erasure_coding::phase1_xor_parity::encode;
///
/// let data = b"HELLO";
/// let encoded = encode(data, 3).unwrap();
/// assert_eq!(encoded.data_chunks.len(), 3);
/// ```
pub fn encode(data: &[u8], num_chunks: usize) -> Result<XorEncoded, XorError> {
    // Validate input
    if data.is_empty() {
        return Err(XorError::EmptyData);
    }
    if num_chunks < 2 {
        return Err(XorError::InvalidChunkCount);
    }

    // Calculate chunk size (with padding if needed)
    let chunk_size = (data.len() + num_chunks - 1) / num_chunks;

    // Split data into chunks
    let mut data_chunks = Vec::with_capacity(num_chunks);
    for i in 0..num_chunks {
        let start = i * chunk_size;

        // Create chunk with padding if we've run out of data
        let mut chunk = if start < data.len() {
            let end = std::cmp::min(start + chunk_size, data.len());
            data[start..end].to_vec()
        } else {
            Vec::new()
        };

        // Pad to chunk_size if needed
        if chunk.len() < chunk_size {
            chunk.resize(chunk_size, 0);
        }

        data_chunks.push(chunk);
    }

    // Generate parity chunk using XOR
    let parity_chunk = generate_parity(&data_chunks);

    Ok(XorEncoded {
        data_chunks,
        parity_chunk,
        chunk_size,
    })
}

/// Generates XOR parity from data chunks
///
/// The parity chunk is computed by XORing all data chunks together.
///
/// # Arguments
///
/// * `chunks` - Slice of data chunks (all must be same size)
///
/// # Returns
///
/// A parity chunk of the same size as input chunks
fn generate_parity(chunks: &[Vec<u8>]) -> Vec<u8> {
    if chunks.is_empty() {
        return Vec::new();
    }

    let chunk_size = chunks[0].len();
    let mut parity = vec![0u8; chunk_size];

    // XOR all chunks together
    for chunk in chunks {
        for (i, &byte) in chunk.iter().enumerate() {
            parity[i] ^= byte;
        }
    }

    parity
}

/// Recovers a lost data chunk using XOR parity
///
/// Uses the property that A ⊕ B ⊕ C ⊕ P = 0, where P is parity.
/// Therefore, if we lose chunk B, we can recover it:
/// B = A ⊕ C ⊕ P
///
/// # Arguments
///
/// * `encoded` - The encoded data structure
/// * `lost_chunk_index` - Index of the chunk to recover (0-based)
///
/// # Returns
///
/// The recovered chunk data
///
/// # Errors
///
/// Returns `XorError::InvalidChunkIndex` if index is out of bounds
///
/// # Example
///
/// ```rust
/// use erasure_coding::phase1_xor_parity::{encode, recover_chunk};
///
/// let data = b"HELLO WORLD";
/// let encoded = encode(data, 3).unwrap();
///
/// // Recover chunk 1
/// let recovered = recover_chunk(&encoded, 1).unwrap();
/// assert_eq!(recovered, encoded.data_chunks[1]);
/// ```
pub fn recover_chunk(encoded: &XorEncoded, lost_chunk_index: usize) -> Result<Vec<u8>, XorError> {
    if lost_chunk_index >= encoded.data_chunks.len() {
        return Err(XorError::InvalidChunkIndex);
    }

    let chunk_size = encoded.chunk_size;
    let mut recovered = vec![0u8; chunk_size];

    // XOR all other data chunks
    for (i, chunk) in encoded.data_chunks.iter().enumerate() {
        if i != lost_chunk_index {
            for (j, &byte) in chunk.iter().enumerate() {
                recovered[j] ^= byte;
            }
        }
    }

    // XOR with parity chunk
    for (i, &byte) in encoded.parity_chunk.iter().enumerate() {
        recovered[i] ^= byte;
    }

    Ok(recovered)
}

/// Decodes the original data from encoded chunks
///
/// # Arguments
///
/// * `encoded` - The encoded data structure
/// * `original_size` - Original data size (to remove padding)
///
/// # Returns
///
/// The original data
pub fn decode(encoded: &XorEncoded, original_size: usize) -> Vec<u8> {
    let mut data = Vec::with_capacity(original_size);

    for chunk in &encoded.data_chunks {
        data.extend_from_slice(chunk);
    }

    // Truncate to original size (remove padding)
    data.truncate(original_size);
    data
}

/// Formats a byte as binary string
pub fn byte_to_binary(byte: u8) -> String {
    format!("{:08b}", byte)
}

/// Formats a chunk as binary string (space-separated bytes)
pub fn chunk_to_binary(chunk: &[u8]) -> String {
    chunk
        .iter()
        .map(|b| byte_to_binary(*b))
        .collect::<Vec<_>>()
        .join(" ")
}

/// Formats a chunk as ASCII string (replacing non-printable with '.')
pub fn chunk_to_ascii(chunk: &[u8]) -> String {
    chunk
        .iter()
        .map(|&b| {
            if b >= 32 && b <= 126 {
                b as char
            } else {
                '.'
            }
        })
        .collect()
}

/// Prints detailed encoding information
pub fn print_encoding_details(encoded: &XorEncoded, original_data: &[u8]) {
    println!("Encoding Process:");
    println!("-----------------");
    println!("Original Data: \"{}\" ({} bytes)", String::from_utf8_lossy(original_data), original_data.len());
    println!();

    for (i, chunk) in encoded.data_chunks.iter().enumerate() {
        println!("Chunk {}: \"{}\"  ({} bytes)", i, chunk_to_ascii(chunk), chunk.len());
        println!("  Binary: {}", chunk_to_binary(chunk));
    }

    println!();
    println!("Parity Chunk (XOR of all chunks):");
    println!("  Binary: {}", chunk_to_binary(&encoded.parity_chunk));
    println!("  ASCII:  \"{}\"", chunk_to_ascii(&encoded.parity_chunk));
    println!();

    let total_chunks = encoded.data_chunks.len() + 1;
    let storage_overhead = ((total_chunks as f64 - encoded.data_chunks.len() as f64) / encoded.data_chunks.len() as f64) * 100.0;
    println!("Total storage: {} chunks (original {} + 1 parity)", total_chunks, encoded.data_chunks.len());
    println!("Storage overhead: {:.1}%", storage_overhead);
}

/// Demonstrates recovery of a specific chunk
pub fn demonstrate_recovery(encoded: &XorEncoded, lost_index: usize) -> Result<(), XorError> {
    println!();
    println!("Recovery Demonstration:");
    println!("-----------------------");
    println!("Simulating loss of Chunk {}...", lost_index);
    println!();

    let original_chunk = &encoded.data_chunks[lost_index];
    let recovered = recover_chunk(encoded, lost_index)?;

    println!("Recovery calculation:");
    print!("  ");
    for (i, _) in encoded.data_chunks.iter().enumerate() {
        if i != lost_index {
            print!("Chunk{} XOR ", i);
        }
    }
    println!("Parity");

    // Show binary calculation for first byte
    if !recovered.is_empty() {
        print!("  = ");
        for (i, chunk) in encoded.data_chunks.iter().enumerate() {
            if i != lost_index && !chunk.is_empty() {
                print!("{} XOR ", byte_to_binary(chunk[0]));
            }
        }
        if !encoded.parity_chunk.is_empty() {
            println!("{} (first byte)", byte_to_binary(encoded.parity_chunk[0]));
        }
        println!("  = {} (ASCII: '{}')", byte_to_binary(recovered[0]), recovered[0] as char);
    }

    println!();
    println!("Recovered Chunk {}: \"{}\"", lost_index, chunk_to_ascii(&recovered));

    if recovered == *original_chunk {
        println!("✓ SUCCESS! Recovery matches original chunk.");
    } else {
        println!("✗ FAILURE! Recovery does not match.");
    }

    println!();
    println!("Original data fully reconstructed!");

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encode_basic() {
        let data = b"HELLO WORLD";
        let encoded = encode(data, 3).unwrap();

        assert_eq!(encoded.data_chunks.len(), 3);
        assert_eq!(encoded.chunk_size, 4); // 11 bytes / 3 chunks = 4 bytes per chunk
    }

    #[test]
    fn test_encode_empty_data() {
        let data = b"";
        let result = encode(data, 3);
        assert_eq!(result, Err(XorError::EmptyData));
    }

    #[test]
    fn test_encode_invalid_chunk_count() {
        let data = b"HELLO";
        let result = encode(data, 1);
        assert_eq!(result, Err(XorError::InvalidChunkCount));
    }

    #[test]
    fn test_recover_chunk_all_positions() {
        let data = b"TEST DATA FOR RECOVERY";
        let num_chunks = 4;
        let encoded = encode(data, num_chunks).unwrap();

        // Test recovery of each chunk position
        for i in 0..num_chunks {
            let recovered = recover_chunk(&encoded, i).unwrap();
            assert_eq!(
                recovered, encoded.data_chunks[i],
                "Failed to recover chunk {}",
                i
            );
        }
    }

    #[test]
    fn test_recover_invalid_index() {
        let data = b"HELLO";
        let encoded = encode(data, 3).unwrap();
        let result = recover_chunk(&encoded, 5);
        assert_eq!(result, Err(XorError::InvalidChunkIndex));
    }

    #[test]
    fn test_encode_decode_roundtrip() {
        let data = b"The quick brown fox jumps over the lazy dog";
        let original_size = data.len();
        let encoded = encode(data, 5).unwrap();
        let decoded = decode(&encoded, original_size);

        assert_eq!(decoded, data);
    }

    #[test]
    fn test_edge_case_single_byte() {
        let data = b"A";
        let encoded = encode(data, 2).unwrap();

        let recovered = recover_chunk(&encoded, 0).unwrap();
        assert_eq!(recovered, encoded.data_chunks[0]);
    }

    #[test]
    fn test_edge_case_exact_division() {
        // 12 bytes should divide evenly into 3 chunks of 4 bytes each
        let data = b"EXACT12BYTES";
        let encoded = encode(data, 3).unwrap();

        assert_eq!(encoded.chunk_size, 4);
        for chunk in &encoded.data_chunks {
            assert_eq!(chunk.len(), 4);
        }
    }

    #[test]
    fn test_parity_property() {
        // XOR property: all chunks XOR parity should equal 0
        let data = b"PARITY TEST";
        let encoded = encode(data, 3).unwrap();

        let mut xor_result = vec![0u8; encoded.chunk_size];

        // XOR all data chunks
        for chunk in &encoded.data_chunks {
            for (i, &byte) in chunk.iter().enumerate() {
                xor_result[i] ^= byte;
            }
        }

        // XOR with parity
        for (i, &byte) in encoded.parity_chunk.iter().enumerate() {
            xor_result[i] ^= byte;
        }

        // Result should be all zeros
        assert_eq!(xor_result, vec![0u8; encoded.chunk_size]);
    }

    #[test]
    fn test_binary_formatting() {
        assert_eq!(byte_to_binary(0b10101010), "10101010");
        assert_eq!(byte_to_binary(0b00000000), "00000000");
        assert_eq!(byte_to_binary(0b11111111), "11111111");
    }

    #[test]
    fn test_ascii_formatting() {
        let chunk = b"Hello!";
        assert_eq!(chunk_to_ascii(chunk), "Hello!");

        let chunk_with_nonprintable = b"Hi\x00\x01!";
        assert_eq!(chunk_to_ascii(chunk_with_nonprintable), "Hi..!");
    }

    #[test]
    fn test_various_chunk_counts() {
        let data = b"Testing with different chunk counts";

        for num_chunks in 2..=10 {
            let encoded = encode(data, num_chunks).unwrap();
            assert_eq!(encoded.data_chunks.len(), num_chunks);

            // Verify recovery works for each configuration
            for i in 0..num_chunks {
                let recovered = recover_chunk(&encoded, i).unwrap();
                assert_eq!(recovered, encoded.data_chunks[i]);
            }
        }
    }
}
