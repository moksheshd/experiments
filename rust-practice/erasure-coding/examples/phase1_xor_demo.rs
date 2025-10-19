//! # Phase 1: XOR-Based Parity - Interactive Demo
//!
//! This interactive demo lets you experiment with XOR-based erasure coding.
//! You can input your own data, choose the number of chunks, and see
//! the encoding and recovery process in action.
//!
//! Run with: cargo run --example phase1_xor_demo

use erasure_coding::phase1_xor_parity::*;
use std::io::{self, Write};

fn main() {
    println!("╔═══════════════════════════════════════════════════════════════╗");
    println!("║  Erasure Coding - Phase 1: XOR-Based Parity                  ║");
    println!("╚═══════════════════════════════════════════════════════════════╝");
    println!();

    // Get user input for data
    print!("Enter your data (or press Enter for default \"HELLO WORLD\"): ");
    io::stdout().flush().unwrap();

    let mut input = String::new();
    io::stdin().read_line(&mut input).unwrap();
    let data = if input.trim().is_empty() {
        "HELLO WORLD"
    } else {
        input.trim()
    };

    // Get number of chunks
    print!("Number of data chunks (2-10, default 3): ");
    io::stdout().flush().unwrap();

    let mut chunks_input = String::new();
    io::stdin().read_line(&mut chunks_input).unwrap();
    let num_chunks = chunks_input
        .trim()
        .parse::<usize>()
        .unwrap_or(3)
        .clamp(2, 10);

    println!();

    // Encode the data
    let original_data = data.as_bytes();
    let encoded = match encode(original_data, num_chunks) {
        Ok(enc) => enc,
        Err(e) => {
            eprintln!("Error encoding data: {}", e);
            return;
        }
    };

    // Display encoding details
    print_encoding_details(&encoded, original_data);

    // Demonstrate recovery for all chunks
    println!();
    println!("═══════════════════════════════════════════════════════════════");
    println!("Testing Recovery for All Chunks");
    println!("═══════════════════════════════════════════════════════════════");

    for i in 0..encoded.data_chunks.len() {
        println!();
        println!("─────────────────────────────────────────────────────────────");
        if let Err(e) = demonstrate_recovery(&encoded, i) {
            eprintln!("Error during recovery: {}", e);
        }
    }

    // Interactive mode: let user choose which chunk to recover
    println!();
    println!("═══════════════════════════════════════════════════════════════");
    println!("Interactive Recovery Mode");
    println!("═══════════════════════════════════════════════════════════════");
    println!();

    loop {
        print!("Which chunk would you like to simulate losing? (0-{}, or 'q' to quit): ", num_chunks - 1);
        io::stdout().flush().unwrap();

        let mut choice = String::new();
        io::stdin().read_line(&mut choice).unwrap();
        let choice = choice.trim();

        if choice == "q" || choice == "quit" {
            break;
        }

        match choice.parse::<usize>() {
            Ok(index) if index < num_chunks => {
                if let Err(e) = demonstrate_recovery(&encoded, index) {
                    eprintln!("Error: {}", e);
                }
            }
            _ => {
                println!("Invalid choice. Please enter a number between 0 and {}.", num_chunks - 1);
            }
        }
        println!();
    }

    println!();
    println!("═══════════════════════════════════════════════════════════════");
    println!("Key Takeaways - Phase 1");
    println!("═══════════════════════════════════════════════════════════════");
    println!();
    println!("✓ XOR is its own inverse: A ⊕ B ⊕ B = A");
    println!("✓ Order doesn't matter: A ⊕ B ⊕ C = C ⊕ A ⊕ B");
    println!("✓ Can recover from ANY single chunk failure");
    println!("✓ Storage overhead: 1 parity chunk for N data chunks");
    println!("✓ This is the foundation of RAID-5 and all erasure codes!");
    println!();
    println!("Next steps:");
    println!("  - Try with different data and chunk counts");
    println!("  - Question: Can we recover from TWO lost chunks? Why not?");
    println!("  - Move to Phase 2 to learn about double parity (RAID-6)");
    println!();
    println!("Thank you for exploring Phase 1!");
}
