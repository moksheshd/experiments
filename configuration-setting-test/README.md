# PostgreSQL User Insertion with Go

This Go code demonstrates how to insert a user into a PostgreSQL database while setting a local application context variable (`app.users_id`) before the insertion.

## Features

- Sets `app.users_id` to 3 before inserting a user
- Uses transactions to ensure data integrity
- Properly handles errors with rollback
- Returns the newly created user with the assigned ID

## Prerequisites

- Go 1.16 or higher
- PostgreSQL database with the schema defined in `postgres_tables.sql`
- PostgreSQL driver for Go

## Setup

1. Initialize a Go module in your project directory:

```bash
go mod init your-module-name
```

2. Install the PostgreSQL driver:

```bash
go mod tidy
```

This will automatically find and install the required PostgreSQL driver.

3. Update the database connection string in `user_insert.go` if needed:

```go
connStr := "postgres://postgres:postgres@localhost/test?sslmode=disable"
```

Replace `postgres` (username), `postgres` (password), `localhost`, and `test` (database name) with your actual PostgreSQL credentials and database name if different.

## Usage

Run the example code:

```bash
go run user_insert.go
```

Or import the `InsertUser` function into your own code:

```go
import "yourmodule/path"

// ...

user, err := InsertUser(db, "John", "Doe")
if err != nil {
    // Handle error
}
fmt.Printf("User created with ID: %d\n", user.ID)
```

## How It Works

1. The code begins a transaction
2. Sets the local variable `app.users_id` to 3 using `SET LOCAL app.users_id = 3`
3. Inserts the user into the `users` table
4. Commits the transaction if successful, or rolls back if any errors occur

This ensures that the `log_user_changes()` trigger function in PostgreSQL will use the value 3 as the `trigerred_by` when logging to the `audit_log` table.
