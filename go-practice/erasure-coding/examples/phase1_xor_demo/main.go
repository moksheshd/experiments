// Phase 1: XOR-Based Parity - Interactive Demo
//
// This interactive demo lets you experiment with XOR-based erasure coding.
// You can input your own data, choose the number of chunks, and see
// the encoding and recovery process in action.
//
// Run with: go run ./examples/phase1_xor_demo
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mokshesh/go-practice/erasure-coding/pkg/erasurecoding/phase1"
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Erasure Coding - Phase 1: XOR-Based Parity                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	// Get user input for data
	fmt.Print("Enter your data (or press Enter for default \"HELLO WORLD\"): ")
	scanner.Scan()
	data := scanner.Text()
	if strings.TrimSpace(data) == "" {
		data = "HELLO WORLD"
	}

	// Get number of chunks
	fmt.Print("Number of data chunks (2-10, default 3): ")
	scanner.Scan()
	chunksInput := scanner.Text()
	numChunks := 3
	if strings.TrimSpace(chunksInput) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(chunksInput))
		if err == nil && parsed >= 2 && parsed <= 10 {
			numChunks = parsed
		}
	}

	fmt.Println()

	// Encode the data
	originalData := []byte(data)
	encoded, err := phase1.Encode(originalData, numChunks)
	if err != nil {
		log.Fatalf("Error encoding data: %v\n", err)
	}

	// Display encoding details
	phase1.PrintEncodingDetails(encoded, originalData)

	// Demonstrate recovery for all chunks
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Testing Recovery for All Chunks")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	for i := 0; i < len(encoded.DataChunks); i++ {
		fmt.Println()
		fmt.Println("─────────────────────────────────────────────────────────────")
		if err := phase1.DemonstrateRecovery(encoded, i); err != nil {
			log.Printf("Error during recovery: %v\n", err)
		}
	}

	// Interactive mode: let user choose which chunk to recover
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Interactive Recovery Mode")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for {
		fmt.Printf("Which chunk would you like to simulate losing? (0-%d, or 'q' to quit): ", numChunks-1)
		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		if choice == "q" || choice == "quit" {
			break
		}

		index, err := strconv.Atoi(choice)
		if err != nil || index < 0 || index >= numChunks {
			fmt.Printf("Invalid choice. Please enter a number between 0 and %d.\n", numChunks-1)
			continue
		}

		if err := phase1.DemonstrateRecovery(encoded, index); err != nil {
			log.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Key Takeaways - Phase 1")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("✓ XOR is its own inverse: A ⊕ B ⊕ B = A")
	fmt.Println("✓ Order doesn't matter: A ⊕ B ⊕ C = C ⊕ A ⊕ B")
	fmt.Println("✓ Can recover from ANY single chunk failure")
	fmt.Println("✓ Storage overhead: 1 parity chunk for N data chunks")
	fmt.Println("✓ This is the foundation of RAID-5 and all erasure codes!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  - Try with different data and chunk counts")
	fmt.Println("  - Question: Can we recover from TWO lost chunks? Why not?")
	fmt.Println("  - Move to Phase 2 to learn about double parity (RAID-6)")
	fmt.Println()
	fmt.Println("Thank you for exploring Phase 1!")
}
