use std::io::{self, Write};

fn main() {
    println!("╔═══════════════════════════════════════════════════════════════╗");
    println!("║         Erasure Coding Training Program                      ║");
    println!("╚═══════════════════════════════════════════════════════════════╝");
    println!();
    println!("Select a phase to explore:");
    println!();
    println!("  1. Phase 1: XOR-Based Parity (RAID-5 style)");
    println!("  2. Phase 2: Double Parity (RAID-6) - Coming soon!");
    println!("  3. Phase 3: Reed-Solomon Fundamentals - Coming soon!");
    println!("  4. Phase 4: Optimized Reed-Solomon - Coming soon!");
    println!("  5. Phase 5: Advanced Topics - Coming soon!");
    println!();
    print!("Enter your choice (1-5, or 'q' to quit): ");
    io::stdout().flush().unwrap();

    let mut choice = String::new();
    io::stdin().read_line(&mut choice).unwrap();

    match choice.trim() {
        "1" => {
            println!();
            println!("Phase 1 selected!");
            println!("Run the interactive demo with:");
            println!("  cargo run --example phase1_xor_demo");
            println!();
            println!("Or run the tests with:");
            println!("  cargo test phase1");
        }
        "2" | "3" | "4" | "5" => {
            println!();
            println!("This phase is not yet implemented.");
            println!("Check the README.md for the training curriculum.");
        }
        "q" | "quit" => {
            println!("Goodbye!");
        }
        _ => {
            println!("Invalid choice. Please run again and select 1-5.");
        }
    }
}
