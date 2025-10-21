package phase1

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncode_Basic(t *testing.T) {
	data := []byte("HELLO WORLD")
	encoded, err := Encode(data, 3)

	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if len(encoded.DataChunks) != 3 {
		t.Errorf("Encode() got %d chunks, want 3", len(encoded.DataChunks))
	}

	expectedChunkSize := (len(data) + 3 - 1) / 3 // 11 bytes / 3 chunks = 4 bytes per chunk
	if encoded.ChunkSize != expectedChunkSize {
		t.Errorf("Encode() chunk size = %d, want %d", encoded.ChunkSize, expectedChunkSize)
	}
}

func TestEncode_EmptyData(t *testing.T) {
	_, err := Encode([]byte{}, 3)
	if err != ErrEmptyData {
		t.Errorf("Encode(empty data) error = %v, want %v", err, ErrEmptyData)
	}
}

func TestEncode_InvalidChunkCount(t *testing.T) {
	data := []byte("HELLO")
	_, err := Encode(data, 1)
	if err != ErrInvalidChunkCount {
		t.Errorf("Encode(chunk count=1) error = %v, want %v", err, ErrInvalidChunkCount)
	}
}

func TestRecoverChunk_AllPositions(t *testing.T) {
	data := []byte("TEST DATA FOR RECOVERY")
	numChunks := 4
	encoded, err := Encode(data, numChunks)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Test recovery of each chunk position
	for i := 0; i < numChunks; i++ {
		t.Run(fmt.Sprintf("Chunk %d", i), func(t *testing.T) {
			recovered, err := RecoverChunk(encoded, i)
			if err != nil {
				t.Errorf("RecoverChunk(%d) error = %v", i, err)
			}
			if !bytes.Equal(recovered, encoded.DataChunks[i]) {
				t.Errorf("RecoverChunk(%d) failed to recover correctly", i)
			}
		})
	}
}

func TestRecoverChunk_InvalidIndex(t *testing.T) {
	data := []byte("HELLO")
	encoded, _ := Encode(data, 3)
	_, err := RecoverChunk(encoded, 5)
	if err != ErrInvalidChunkIndex {
		t.Errorf("RecoverChunk(invalid index) error = %v, want %v", err, ErrInvalidChunkIndex)
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		numChunks  int
	}{
		{"short string", []byte("The quick brown fox jumps over the lazy dog"), 5},
		{"exact division", []byte("EXACT12BYTES"), 3},
		{"single byte", []byte("A"), 2},
		{"long string", []byte("This is a much longer string that should be split into many chunks for testing"), 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSize := len(tt.data)
			encoded, err := Encode(tt.data, tt.numChunks)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			decoded := Decode(encoded, originalSize)
			if !bytes.Equal(decoded, tt.data) {
				t.Errorf("Decode() = %q, want %q", decoded, tt.data)
			}
		})
	}
}

func TestParityProperty(t *testing.T) {
	// XOR property: all chunks XOR parity should equal 0
	data := []byte("PARITY TEST")
	encoded, err := Encode(data, 3)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	xorResult := make([]byte, encoded.ChunkSize)

	// XOR all data chunks
	for _, chunk := range encoded.DataChunks {
		for i, b := range chunk {
			xorResult[i] ^= b
		}
	}

	// XOR with parity
	for i, b := range encoded.ParityChunk {
		xorResult[i] ^= b
	}

	// Result should be all zeros
	expectedZeros := make([]byte, encoded.ChunkSize)
	if !bytes.Equal(xorResult, expectedZeros) {
		t.Errorf("Parity property failed: XOR result is not all zeros")
	}
}

func TestByteToBinary(t *testing.T) {
	tests := []struct {
		input    byte
		expected string
	}{
		{0b10101010, "10101010"},
		{0b00000000, "00000000"},
		{0b11111111, "11111111"},
		{0b00000001, "00000001"},
		{65, "01000001"}, // 'A'
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("byte_%d", tt.input), func(t *testing.T) {
			got := ByteToBinary(tt.input)
			if got != tt.expected {
				t.Errorf("ByteToBinary(%d) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestChunkToASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"printable", []byte("Hello!"), "Hello!"},
		{"with non-printable", []byte("Hi\x00\x01!"), "Hi..!"},
		{"all non-printable", []byte{0, 1, 2}, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChunkToASCII(tt.input)
			if got != tt.expected {
				t.Errorf("ChunkToASCII() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestVariousChunkCounts(t *testing.T) {
	data := []byte("Testing with different chunk counts")

	for numChunks := 2; numChunks <= 10; numChunks++ {
		t.Run(fmt.Sprintf("chunks_%d", numChunks), func(t *testing.T) {
			encoded, err := Encode(data, numChunks)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			if len(encoded.DataChunks) != numChunks {
				t.Errorf("Encode() got %d chunks, want %d", len(encoded.DataChunks), numChunks)
			}

			// Verify recovery works for each configuration
			for i := 0; i < numChunks; i++ {
				recovered, err := RecoverChunk(encoded, i)
				if err != nil {
					t.Errorf("RecoverChunk(%d) error = %v", i, err)
				}
				if !bytes.Equal(recovered, encoded.DataChunks[i]) {
					t.Errorf("RecoverChunk(%d) failed for %d chunks configuration", i, numChunks)
				}
			}
		})
	}
}

func TestEdgeCase_ExactDivision(t *testing.T) {
	// 12 bytes should divide evenly into 3 chunks of 4 bytes each
	data := []byte("EXACT12BYTES")
	encoded, err := Encode(data, 3)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if encoded.ChunkSize != 4 {
		t.Errorf("ChunkSize = %d, want 4", encoded.ChunkSize)
	}

	for _, chunk := range encoded.DataChunks {
		if len(chunk) != 4 {
			t.Errorf("Chunk length = %d, want 4", len(chunk))
		}
	}
}

func TestEdgeCase_SingleByte(t *testing.T) {
	data := []byte("A")
	encoded, err := Encode(data, 2)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	recovered, err := RecoverChunk(encoded, 0)
	if err != nil {
		t.Fatalf("RecoverChunk() error = %v", err)
	}

	if !bytes.Equal(recovered, encoded.DataChunks[0]) {
		t.Errorf("RecoverChunk() failed for single byte")
	}
}

// Benchmark encoding
func BenchmarkEncode(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 1000) // ~15KB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = Encode(data, 5)
	}
}

// Benchmark recovery
func BenchmarkRecoverChunk(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 1000) // ~15KB
	encoded, _ := Encode(data, 5)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = RecoverChunk(encoded, 2)
	}
}

// Example demonstrates basic usage
func ExampleEncode() {
	data := []byte("HELLO WORLD")
	encoded, err := Encode(data, 3)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Data chunks: %d\n", len(encoded.DataChunks))
	fmt.Printf("Parity chunk: %d bytes\n", len(encoded.ParityChunk))
	// Output:
	// Data chunks: 3
	// Parity chunk: 4 bytes
}

// Example demonstrates chunk recovery
func ExampleRecoverChunk() {
	data := []byte("HELLO WORLD")
	encoded, _ := Encode(data, 3)

	// Simulate losing chunk 1
	recovered, err := RecoverChunk(encoded, 1)
	if err != nil {
		panic(err)
	}

	if bytes.Equal(recovered, encoded.DataChunks[1]) {
		fmt.Println("Recovery successful!")
	}
	// Output:
	// Recovery successful!
}
