// Package phase1 implements XOR-based parity erasure coding (RAID-5 style).
//
// This is the simplest form of erasure coding using XOR operations.
// It demonstrates RAID-5 style single parity protection.
//
// Key Concepts:
//   - Split data into N chunks
//   - Generate 1 parity chunk using XOR of all data chunks
//   - Can recover from any single chunk failure
//   - XOR is reversible: A ⊕ B ⊕ B = A
//
// Example:
//
//	data := []byte("HELLO WORLD")
//	encoded, err := Encode(data, 3)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Simulate losing chunk 1 and recover it
//	recovered, err := RecoverChunk(encoded, 1)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// recovered should equal encoded.DataChunks[1]
package phase1

import (
	"bytes"
	"fmt"
	"strings"
)

// XorEncoded represents encoded data with XOR parity
type XorEncoded struct {
	// Original data chunks
	DataChunks [][]byte
	// XOR parity chunk (same size as each data chunk)
	ParityChunk []byte
	// Size of each chunk in bytes
	ChunkSize int
}

// XorError represents errors that can occur during encoding or recovery
type XorError struct {
	message string
}

func (e *XorError) Error() string {
	return e.message
}

// Common errors
var (
	ErrEmptyData         = &XorError{"input data cannot be empty"}
	ErrInvalidChunkCount = &XorError{"number of chunks must be at least 2"}
	ErrInvalidChunkIndex = &XorError{"chunk index is out of bounds"}
)

// Encode splits data into chunks and generates XOR parity
//
// Arguments:
//   - data: The input data to encode
//   - numChunks: Number of data chunks to split into (must be >= 2)
//
// Returns an XorEncoded structure containing the data chunks and parity chunk
//
// Errors:
//   - ErrEmptyData if input is empty
//   - ErrInvalidChunkCount if numChunks < 2
func Encode(data []byte, numChunks int) (*XorEncoded, error) {
	// Validate input
	if len(data) == 0 {
		return nil, ErrEmptyData
	}
	if numChunks < 2 {
		return nil, ErrInvalidChunkCount
	}

	// Calculate chunk size (with padding if needed)
	chunkSize := (len(data) + numChunks - 1) / numChunks

	// Split data into chunks
	dataChunks := make([][]byte, numChunks)
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize

		// Create chunk with padding if we've run out of data
		var chunk []byte
		if start < len(data) {
			end := start + chunkSize
			if end > len(data) {
				end = len(data)
			}
			chunk = make([]byte, chunkSize)
			copy(chunk, data[start:end])
		} else {
			chunk = make([]byte, chunkSize)
		}

		dataChunks[i] = chunk
	}

	// Generate parity chunk using XOR
	parityChunk := generateParity(dataChunks)

	return &XorEncoded{
		DataChunks:  dataChunks,
		ParityChunk: parityChunk,
		ChunkSize:   chunkSize,
	}, nil
}

// generateParity computes XOR parity from data chunks
//
// The parity chunk is computed by XORing all data chunks together.
func generateParity(chunks [][]byte) []byte {
	if len(chunks) == 0 {
		return []byte{}
	}

	chunkSize := len(chunks[0])
	parity := make([]byte, chunkSize)

	// XOR all chunks together
	for _, chunk := range chunks {
		for i, b := range chunk {
			parity[i] ^= b
		}
	}

	return parity
}

// RecoverChunk recovers a lost data chunk using XOR parity
//
// Uses the property that A ⊕ B ⊕ C ⊕ P = 0, where P is parity.
// Therefore, if we lose chunk B, we can recover it: B = A ⊕ C ⊕ P
//
// Arguments:
//   - encoded: The encoded data structure
//   - lostChunkIndex: Index of the chunk to recover (0-based)
//
// Returns the recovered chunk data
//
// Errors:
//   - ErrInvalidChunkIndex if index is out of bounds
func RecoverChunk(encoded *XorEncoded, lostChunkIndex int) ([]byte, error) {
	if lostChunkIndex >= len(encoded.DataChunks) {
		return nil, ErrInvalidChunkIndex
	}

	chunkSize := encoded.ChunkSize
	recovered := make([]byte, chunkSize)

	// XOR all other data chunks
	for i, chunk := range encoded.DataChunks {
		if i != lostChunkIndex {
			for j, b := range chunk {
				recovered[j] ^= b
			}
		}
	}

	// XOR with parity chunk
	for i, b := range encoded.ParityChunk {
		recovered[i] ^= b
	}

	return recovered, nil
}

// Decode reconstructs the original data from encoded chunks
//
// Arguments:
//   - encoded: The encoded data structure
//   - originalSize: Original data size (to remove padding)
//
// Returns the original data
func Decode(encoded *XorEncoded, originalSize int) []byte {
	data := make([]byte, 0, originalSize)

	for _, chunk := range encoded.DataChunks {
		data = append(data, chunk...)
	}

	// Truncate to original size (remove padding)
	if len(data) > originalSize {
		data = data[:originalSize]
	}

	return data
}

// ByteToBinary formats a byte as binary string
func ByteToBinary(b byte) string {
	return fmt.Sprintf("%08b", b)
}

// ChunkToBinary formats a chunk as binary string (space-separated bytes)
func ChunkToBinary(chunk []byte) string {
	var parts []string
	for _, b := range chunk {
		parts = append(parts, ByteToBinary(b))
	}
	return strings.Join(parts, " ")
}

// ChunkToASCII formats a chunk as ASCII string (replacing non-printable with '.')
func ChunkToASCII(chunk []byte) string {
	var buf bytes.Buffer
	for _, b := range chunk {
		if b >= 32 && b <= 126 {
			buf.WriteByte(b)
		} else {
			buf.WriteByte('.')
		}
	}
	return buf.String()
}

// PrintEncodingDetails prints detailed encoding information
func PrintEncodingDetails(encoded *XorEncoded, originalData []byte) {
	fmt.Println("Encoding Process:")
	fmt.Println("-----------------")
	fmt.Printf("Original Data: %q (%d bytes)\n", string(originalData), len(originalData))
	fmt.Println()

	for i, chunk := range encoded.DataChunks {
		fmt.Printf("Chunk %d: %q  (%d bytes)\n", i, ChunkToASCII(chunk), len(chunk))
		fmt.Printf("  Binary: %s\n", ChunkToBinary(chunk))
	}

	fmt.Println()
	fmt.Println("Parity Chunk (XOR of all chunks):")
	fmt.Printf("  Binary: %s\n", ChunkToBinary(encoded.ParityChunk))
	fmt.Printf("  ASCII:  %q\n", ChunkToASCII(encoded.ParityChunk))
	fmt.Println()

	totalChunks := len(encoded.DataChunks) + 1
	storageOverhead := (float64(totalChunks) - float64(len(encoded.DataChunks))) / float64(len(encoded.DataChunks)) * 100.0
	fmt.Printf("Total storage: %d chunks (original %d + 1 parity)\n", totalChunks, len(encoded.DataChunks))
	fmt.Printf("Storage overhead: %.1f%%\n", storageOverhead)
}

// DemonstrateRecovery demonstrates recovery of a specific chunk
func DemonstrateRecovery(encoded *XorEncoded, lostIndex int) error {
	fmt.Println()
	fmt.Println("Recovery Demonstration:")
	fmt.Println("-----------------------")
	fmt.Printf("Simulating loss of Chunk %d...\n", lostIndex)
	fmt.Println()

	originalChunk := encoded.DataChunks[lostIndex]
	recovered, err := RecoverChunk(encoded, lostIndex)
	if err != nil {
		return err
	}

	fmt.Println("Recovery calculation:")
	fmt.Print("  ")
	for i := range encoded.DataChunks {
		if i != lostIndex {
			fmt.Printf("Chunk%d XOR ", i)
		}
	}
	fmt.Println("Parity")

	// Show binary calculation for first byte
	if len(recovered) > 0 {
		fmt.Print("  = ")
		for i, chunk := range encoded.DataChunks {
			if i != lostIndex && len(chunk) > 0 {
				fmt.Printf("%s XOR ", ByteToBinary(chunk[0]))
			}
		}
		if len(encoded.ParityChunk) > 0 {
			fmt.Printf("%s (first byte)\n", ByteToBinary(encoded.ParityChunk[0]))
		}
		if recovered[0] >= 32 && recovered[0] <= 126 {
			fmt.Printf("  = %s (ASCII: '%c')\n", ByteToBinary(recovered[0]), recovered[0])
		} else {
			fmt.Printf("  = %s\n", ByteToBinary(recovered[0]))
		}
	}

	fmt.Println()
	fmt.Printf("Recovered Chunk %d: %q\n", lostIndex, ChunkToASCII(recovered))

	if bytes.Equal(recovered, originalChunk) {
		fmt.Println("✓ SUCCESS! Recovery matches original chunk.")
	} else {
		fmt.Println("✗ FAILURE! Recovery does not match.")
	}

	fmt.Println()
	fmt.Println("Original data fully reconstructed!")

	return nil
}
