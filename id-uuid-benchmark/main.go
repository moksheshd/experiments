package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// InitDB initializes the database connection and creates tables if they don't exist
func InitDB(host string, port int, user, password, dbname string) error {
	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create the ID table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS id_table (
			id BIGSERIAL PRIMARY KEY,
			age INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create id_table: %w", err)
	}

	// Create the UUID table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS uuid_table (
			uuid UUID PRIMARY KEY,
			age INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create uuid_table: %w", err)
	}

	fmt.Println("Database tables initialized successfully")
	return nil
}

// BenchmarkUUIDTableInsert inserts a specified number of records into the uuid_table and measures the time taken
func BenchmarkUUIDTableInsert(db *sql.DB, totalRecords int, batchSize int) error {
	startTime := time.Now()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Prepare statement
	stmt, err := tx.Prepare("INSERT INTO uuid_table (uuid, age) VALUES ($1, $2)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Initialize random number generator with a source
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Insert records in batches
	for i := 0; i < totalRecords; i++ {
		// Generate a random UUID
		id := uuid.New()
		age := rng.Intn(100) // Random age between 0-99

		_, err := stmt.Exec(id, age)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert record %d: %w", i, err)
		}

		// Commit and start new transaction every batchSize records
		if (i+1)%batchSize == 0 {
			err = tx.Commit()
			if err != nil {
				return fmt.Errorf("failed to commit transaction at record %d: %w", i, err)
			}

			// Report progress
			elapsed := time.Since(startTime)
			recordsInserted := i + 1
			recordsPerSecond := float64(recordsInserted) / elapsed.Seconds()
			percentComplete := float64(recordsInserted) / float64(totalRecords) * 100

			fmt.Printf("Progress: %d/%d records (%.2f%%) | Rate: %.2f records/sec\n",
				recordsInserted, totalRecords, percentComplete, recordsPerSecond)

			// Start new transaction
			tx, err = db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin new transaction at record %d: %w", i, err)
			}

			stmt, err = tx.Prepare("INSERT INTO uuid_table (uuid, age) VALUES ($1, $2)")
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to prepare statement for new transaction at record %d: %w", i, err)
			}
		}
	}

	// Commit any remaining records
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit final transaction: %w", err)
	}

	// Calculate and report final statistics
	totalTime := time.Since(startTime)
	recordsPerSecond := float64(totalRecords) / totalTime.Seconds()

	fmt.Printf("\nBenchmark Results:\n")
	fmt.Printf("Total records inserted: %d\n", totalRecords)
	fmt.Printf("Total time: %v\n", totalTime)
	fmt.Printf("Average insertion rate: %.2f records/sec\n", recordsPerSecond)

	return nil
}

// BenchmarkIDTableInsert inserts a specified number of records into the id_table and measures the time taken
func BenchmarkIDTableInsert(db *sql.DB, totalRecords int, batchSize int) error {
	startTime := time.Now()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Prepare statement
	stmt, err := tx.Prepare("INSERT INTO id_table (age) VALUES ($1)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Initialize random number generator with a source
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Insert records in batches
	for i := 0; i < totalRecords; i++ {
		age := rng.Intn(100) // Random age between 0-99

		_, err := stmt.Exec(age)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert record %d: %w", i, err)
		}

		// Commit and start new transaction every batchSize records
		if (i+1)%batchSize == 0 {
			err = tx.Commit()
			if err != nil {
				return fmt.Errorf("failed to commit transaction at record %d: %w", i, err)
			}

			// Report progress
			elapsed := time.Since(startTime)
			recordsInserted := i + 1
			recordsPerSecond := float64(recordsInserted) / elapsed.Seconds()
			percentComplete := float64(recordsInserted) / float64(totalRecords) * 100

			fmt.Printf("Progress: %d/%d records (%.2f%%) | Rate: %.2f records/sec\n",
				recordsInserted, totalRecords, percentComplete, recordsPerSecond)

			// Start new transaction
			tx, err = db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin new transaction at record %d: %w", i, err)
			}

			stmt, err = tx.Prepare("INSERT INTO id_table (age) VALUES ($1)")
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to prepare statement for new transaction at record %d: %w", i, err)
			}
		}
	}

	// Commit any remaining records
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit final transaction: %w", err)
	}

	// Calculate and report final statistics
	totalTime := time.Since(startTime)
	recordsPerSecond := float64(totalRecords) / totalTime.Seconds()

	fmt.Printf("\nBenchmark Results:\n")
	fmt.Printf("Total records inserted: %d\n", totalRecords)
	fmt.Printf("Total time: %v\n", totalTime)
	fmt.Printf("Average insertion rate: %.2f records/sec\n", recordsPerSecond)

	return nil
}

// TableStats holds statistics about a database table
type TableStats struct {
	TableName        string
	RowCount         int64
	TableSizeBytes   int64
	TableSizePretty  string
	IndexSizeBytes   int64
	IndexSizePretty  string
	TotalSizeBytes   int64
	TotalSizePretty  string
	AvgRowSizeBytes  float64
	IndexToDataRatio float64
}

// GetTableStats retrieves size and index statistics for the specified table
func GetTableStats(db *sql.DB, tableName string) (*TableStats, error) {
	stats := &TableStats{
		TableName: tableName,
	}

	// Get row count
	err := db.QueryRow("SELECT COUNT(*) FROM " + tableName).Scan(&stats.RowCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get row count for %s: %w", tableName, err)
	}

	// Get table size (data only)
	err = db.QueryRow("SELECT pg_table_size($1)", tableName).Scan(&stats.TableSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get table size for %s: %w", tableName, err)
	}

	// Get pretty table size
	err = db.QueryRow("SELECT pg_size_pretty(pg_table_size($1))", tableName).Scan(&stats.TableSizePretty)
	if err != nil {
		return nil, fmt.Errorf("failed to get pretty table size for %s: %w", tableName, err)
	}

	// Get index size
	err = db.QueryRow(`
		SELECT 
			COALESCE(sum(pg_relation_size(indexrelid)), 0)
		FROM pg_index
		WHERE indrelid = $1::regclass
	`, tableName).Scan(&stats.IndexSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get index size for %s: %w", tableName, err)
	}

	// Get pretty index size
	err = db.QueryRow(`
		SELECT 
			pg_size_pretty(COALESCE(sum(pg_relation_size(indexrelid)), 0))
		FROM pg_index
		WHERE indrelid = $1::regclass
	`, tableName).Scan(&stats.IndexSizePretty)
	if err != nil {
		return nil, fmt.Errorf("failed to get pretty index size for %s: %w", tableName, err)
	}

	// Get total size (including indexes and TOAST)
	err = db.QueryRow("SELECT pg_total_relation_size($1)", tableName).Scan(&stats.TotalSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get total size for %s: %w", tableName, err)
	}

	// Get pretty total size
	err = db.QueryRow("SELECT pg_size_pretty(pg_total_relation_size($1))", tableName).Scan(&stats.TotalSizePretty)
	if err != nil {
		return nil, fmt.Errorf("failed to get pretty total size for %s: %w", tableName, err)
	}

	// Calculate average row size (if there are rows)
	if stats.RowCount > 0 {
		stats.AvgRowSizeBytes = float64(stats.TableSizeBytes) / float64(stats.RowCount)
	}

	// Calculate index to data ratio (if table size is not zero)
	if stats.TableSizeBytes > 0 {
		stats.IndexToDataRatio = float64(stats.IndexSizeBytes) / float64(stats.TableSizeBytes)
	}

	return stats, nil
}

// PrintTableStats prints the statistics for a table
func PrintTableStats(stats *TableStats) {
	fmt.Printf("Table: %s\n", stats.TableName)
	fmt.Printf("  Row count: %d\n", stats.RowCount)
	fmt.Printf("  Table size (data only): %s (%d bytes)\n", stats.TableSizePretty, stats.TableSizeBytes)
	fmt.Printf("  Index size: %s (%d bytes)\n", stats.IndexSizePretty, stats.IndexSizeBytes)
	fmt.Printf("  Total size (incl. indexes): %s (%d bytes)\n", stats.TotalSizePretty, stats.TotalSizeBytes)
	fmt.Printf("  Average row size: %.2f bytes\n", stats.AvgRowSizeBytes)
	fmt.Printf("  Index to data ratio: %.2f\n", stats.IndexToDataRatio)
}

// bytesToHumanReadable converts bytes to a human-readable string (KB, MB, GB, etc.)
func bytesToHumanReadable(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// TruncateTable truncates the specified table to start with a clean slate
func TruncateTable(db *sql.DB, tableName string) error {
	_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY", tableName))
	if err != nil {
		return fmt.Errorf("failed to truncate table %s: %w", tableName, err)
	}
	fmt.Printf("Table %s truncated successfully\n", tableName)
	return nil
}

// GetDBConnection returns a database connection using the provided parameters
func GetDBConnection(host string, port int, user, password, dbname string) (*sql.DB, error) {
	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunBenchmarks runs both ID and UUID table benchmarks and compares the results
func RunBenchmarks(db *sql.DB, totalRecords int, batchSize int) error {
	var idTime, uuidTime time.Duration
	var idRate, uuidRate float64

	// Benchmark ID table
	fmt.Printf("\n=== BENCHMARK: ID TABLE ===\n\n")

	// Truncate the id_table to start with a clean slate
	err := TruncateTable(db, "id_table")
	if err != nil {
		return fmt.Errorf("failed to truncate id_table: %w", err)
	}

	// Run the ID table benchmark
	fmt.Printf("Starting benchmark for inserting %d records into id_table...\n", totalRecords)
	startTime := time.Now()
	err = BenchmarkIDTableInsert(db, totalRecords, batchSize)
	if err != nil {
		return fmt.Errorf("id_table benchmark failed: %w", err)
	}
	idTime = time.Since(startTime)
	idRate = float64(totalRecords) / idTime.Seconds()

	// Benchmark UUID table
	fmt.Printf("\n=== BENCHMARK: UUID TABLE ===\n\n")

	// Truncate the uuid_table to start with a clean slate
	err = TruncateTable(db, "uuid_table")
	if err != nil {
		return fmt.Errorf("failed to truncate uuid_table: %w", err)
	}

	// Run the UUID table benchmark
	fmt.Printf("Starting benchmark for inserting %d records into uuid_table...\n", totalRecords)
	startTime = time.Now()
	err = BenchmarkUUIDTableInsert(db, totalRecords, batchSize)
	if err != nil {
		return fmt.Errorf("uuid_table benchmark failed: %w", err)
	}
	uuidTime = time.Since(startTime)
	uuidRate = float64(totalRecords) / uuidTime.Seconds()

	// Get table statistics for ID table
	fmt.Printf("\n=== TABLE STATISTICS: ID TABLE ===\n\n")
	idStats, err := GetTableStats(db, "id_table")
	if err != nil {
		return fmt.Errorf("failed to get id_table statistics: %w", err)
	}
	PrintTableStats(idStats)

	// Get table statistics for UUID table
	fmt.Printf("\n=== TABLE STATISTICS: UUID TABLE ===\n\n")
	uuidStats, err := GetTableStats(db, "uuid_table")
	if err != nil {
		return fmt.Errorf("failed to get uuid_table statistics: %w", err)
	}
	PrintTableStats(uuidStats)

	// Compare results
	fmt.Printf("\n=== BENCHMARK COMPARISON ===\n\n")
	fmt.Printf("Total records inserted: %d\n\n", totalRecords)

	fmt.Printf("ID Table (BIGSERIAL):\n")
	fmt.Printf("  Total time: %v\n", idTime)
	fmt.Printf("  Average insertion rate: %.2f records/sec\n", idRate)
	fmt.Printf("  Table size: %s\n", idStats.TableSizePretty)
	fmt.Printf("  Index size: %s\n", idStats.IndexSizePretty)
	fmt.Printf("  Total size: %s\n", idStats.TotalSizePretty)
	fmt.Printf("  Average row size: %.2f bytes\n", idStats.AvgRowSizeBytes)
	fmt.Printf("  Index to data ratio: %.2f\n\n", idStats.IndexToDataRatio)

	fmt.Printf("UUID Table:\n")
	fmt.Printf("  Total time: %v\n", uuidTime)
	fmt.Printf("  Average insertion rate: %.2f records/sec\n", uuidRate)
	fmt.Printf("  Table size: %s\n", uuidStats.TableSizePretty)
	fmt.Printf("  Index size: %s\n", uuidStats.IndexSizePretty)
	fmt.Printf("  Total size: %s\n", uuidStats.TotalSizePretty)
	fmt.Printf("  Average row size: %.2f bytes\n", uuidStats.AvgRowSizeBytes)
	fmt.Printf("  Index to data ratio: %.2f\n\n", uuidStats.IndexToDataRatio)

	// Calculate performance difference
	var fasterTable string
	var speedDifference float64

	if idRate > uuidRate {
		fasterTable = "ID Table (BIGSERIAL)"
		speedDifference = (idRate / uuidRate) - 1
	} else {
		fasterTable = "UUID Table"
		speedDifference = (uuidRate / idRate) - 1
	}

	// Calculate storage efficiency
	var moreEfficientTable string
	var sizeDifference float64
	var sizeRatio float64

	if idStats.TotalSizeBytes < uuidStats.TotalSizeBytes {
		moreEfficientTable = "ID Table (BIGSERIAL)"
		sizeDifference = float64(uuidStats.TotalSizeBytes - idStats.TotalSizeBytes)
		sizeRatio = float64(uuidStats.TotalSizeBytes) / float64(idStats.TotalSizeBytes)
	} else {
		moreEfficientTable = "UUID Table"
		sizeDifference = float64(idStats.TotalSizeBytes - uuidStats.TotalSizeBytes)
		sizeRatio = float64(idStats.TotalSizeBytes) / float64(uuidStats.TotalSizeBytes)
	}

	fmt.Printf("Performance Comparison:\n")
	fmt.Printf("  %s is %.2f%% faster\n", fasterTable, speedDifference*100)

	fmt.Printf("\nStorage Comparison:\n")
	fmt.Printf("  %s uses %.2f%% less space (%.2f times smaller)\n",
		moreEfficientTable, (1-1/sizeRatio)*100, sizeRatio)
	fmt.Printf("  Absolute difference: %s (%d bytes)\n",
		bytesToHumanReadable(int64(sizeDifference)), int64(sizeDifference))

	return nil
}

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("Go project initialized successfully!")

	// Default values
	host := "172.26.176.1"
	port := 5432
	user := "postgres"
	password := "postgres"
	dbname := "id_uuid_performance"
	totalRecords := 100
	batchSize := 10000

	// Check if a command-line argument is provided for the number of records
	if len(os.Args) > 1 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			totalRecords = n
			fmt.Printf("Using custom record count: %d\n", totalRecords)
		}
	}

	// Initialize database with the provided connection parameters
	err := InitDB(host, port, user, password, dbname)
	if err != nil {
		fmt.Printf("Database initialization failed: %v\n", err)
		return
	}

	// Get a database connection for benchmarking
	db, err := GetDBConnection(host, port, user, password, dbname)
	if err != nil {
		fmt.Printf("Failed to get database connection for benchmarking: %v\n", err)
		return
	}
	defer db.Close()

	// Run both benchmarks and compare results
	err = RunBenchmarks(db, totalRecords, batchSize)
	if err != nil {
		fmt.Printf("Benchmarking failed: %v\n", err)
		return
	}

	fmt.Println("Benchmarks completed successfully")
}
