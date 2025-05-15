package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// User represents a user in the database
type User struct {
	ID        int64
	FirstName string
	LastName  string
}

// InsertUser inserts a new user into the users table
// It sets app.users_id to 3 before insertion to track who performed the operation
func InsertUser(db *sql.DB, firstName, lastName string) (*User, error) {
	// Create a new user object
	user := &User{
		FirstName: firstName,
		LastName:  lastName,
	}

	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback in case anything fails
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Set the local variable app.users_id to 3
	_, err = tx.Exec("SET LOCAL app.users_id = 3")
	if err != nil {
		return nil, fmt.Errorf("failed to set local app.users_id: %w", err)
	}

	// Insert the user
	err = tx.QueryRow(
		"INSERT INTO users (first_name, last_name) VALUES ($1, $2) RETURNING id",
		user.FirstName, user.LastName,
	).Scan(&user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	// Set tx to nil to prevent rollback after successful commit
	tx = nil

	return user, nil
}

// Example usage
func main() {
	// Connection parameters
	connStr := "postgres://postgres:postgres@localhost/test?sslmode=disable"

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Insert a new user
	newUser, err := InsertUser(db, "John", "Doe")
	if err != nil {
		log.Fatalf("Failed to insert user: %v", err)
	}

	fmt.Printf("Successfully inserted user with ID: %d\n", newUser.ID)
}
