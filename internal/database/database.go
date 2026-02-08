package database

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/mpilhlt/embapi/internal/crypto"
	"github.com/mpilhlt/embapi/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database initialization

func InitDB(options *models.Options) (*pgxpool.Pool, error) {
	println("--- Connecting to database ...")

	// urlExample := "postgres://username:password@localhost:5432/database_name"
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		options.DBUser, options.DBPassword, options.DBHost, options.DBPort, options.DBName)

	// Connect to the database, first without concurrency to check the schema version
	ctx_cancel, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := VerifySchema(ctx_cancel, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to verify schema: %v\n", err)
		return nil, err
	}

	// For the actual application, connect to the db using a connection pool
	ctx_bg := context.Background()
	// pool, err := pgxpool.New(ctx_bg, url)

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Set application name for easier identification
	cfg.ConnConfig.RuntimeParams["application_name"] = "DHaMPS-VDB"

	// Configure logging
	// cfg.ConnConfig.Logger = logrusadapter.NewLogger(logger)
	// cfg.ConnConfig.LogLevel = pgxpool.LogLevelDebug

	// Connect to the pool
	pool, err := pgxpool.NewWithConfig(ctx_bg, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to get connection pool from database: %v\n", err)
		return nil, err
	}
	fmt.Printf("    Successfully got connection pool from postgres database: %s@%s:%d/%s.\n", // alternatively, print conn.ConnInfo().DatabaseName
		options.DBUser, options.DBHost, options.DBPort, options.DBName)

	// Run test query (get all users)
	conn, err := pool.Acquire(ctx_cancel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to get connection from pool: %v\n", err)
		return nil, err
	}
	defer conn.Release()
	err = testQuery(ctx_cancel, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to run test query: %v\n", err)
		return nil, err
	}
	// conn.Conn().Close(ctx_cancel)

	// Done, everything has been set up. Return connection pool.
	println("--- Database up and initialized.")
	return pool, nil
}

func VerifySchema(ctx context.Context, url string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to connect to database: %v\n", err)
		return err
	}
	// Check schema version of database
	println("--- Checking schema version of database ...")
	migrator, err := NewMigrator(ctx, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to initialize migrator: %v\n", err)
		return err
	}
	// get the current migration status
	now, exp, info, err := migrator.Info()
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to get schema info: %v\n", err)
		return err
	}
	if now < exp {
		// migration is required, dump out the current state
		// and perform the migration
		println("    Database scheme needs migration, current state: ")
		println(info)

		ctx_cancel, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err = migrator.Migrate(ctx_cancel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "EEE Unable to migrate schema: %v\n", err)
			return err
		}
		println("    Database migration successful, schema up to date!")
		
		// Initialize system user key after migration
		err = InitializeSystemUserKey(ctx, conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "EEE Unable to initialize system user key: %v\n", err)
			return err
		}
	} else {
		println("    Database schema up to date, no database migration needed")
		
		// Still check if system user key needs initialization
		err = InitializeSystemUserKey(ctx, conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "EEE Unable to initialize system user key: %v\n", err)
			return err
		}
	}

	conn.Close(ctx)
	return nil
}

func testQuery(ctx context.Context, conn *pgxpool.Conn) error {
	println("--- Send test query - list all users:")

	ctx_cancel, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	queries := New(conn)
	users, err := queries.GetAllUsers(ctx_cancel, GetAllUsersParams{Limit: 10, Offset: 0})
	if err != nil {
		fmt.Fprintf(os.Stderr, "EEE Unable to get users: %v\n", err)
		return err
	}
	for _, u := range users {
		fmt.Printf("    User: %v\n", u)
	}
	return nil
}

// WithTransaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
// The context is given a timeout of 10 seconds for the transaction.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	// Create a context with timeout for the transaction
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Begin the transaction
	tx, err := pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback is called if we don't commit
	// This is safe to call even after commit
	defer func() {
		if err := tx.Rollback(txCtx); err != nil && err != pgx.ErrTxClosed {
			fmt.Fprintf(os.Stderr, "EEE Failed to rollback transaction: %v\n", err)
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// InitializeSystemUserKey replaces the placeholder system user API key with a secure key
// This should be called after migrations are complete
func InitializeSystemUserKey(ctx context.Context, conn *pgx.Conn) error {
	// Get the encryption key from environment
	encKey, err := crypto.GetEncryptionKeyFromEnv()
	if err != nil {
		// Don't expose the key in error messages
		return fmt.Errorf("ENCRYPTION_KEY environment variable not set")
	}

	// Check if _system user has the placeholder key
	var currentKey string
	err = conn.QueryRow(ctx, 
		"SELECT embapi_key FROM users WHERE user_handle = '_system'").Scan(&currentKey)
	if err != nil {
		// User doesn't exist or other error, skip silently
		return nil
	}

	// If it's the placeholder, generate a new key using the encryption key
	if currentKey == "0000000000000000000000000000000000000000000000000000000000000000" {
		// Generate a deterministic key from the encryption key
		// We use the encryption key to encrypt a known string, then hash it
		// This ensures the same key is generated each time with the same ENCRYPTION_KEY
		knownString := "_system_user_api_key"
		encrypted, err := encKey.Encrypt(knownString)
		if err != nil {
			// Don't expose the key in error messages
			return fmt.Errorf("failed to generate system user API key")
		}
		
		// Convert encrypted bytes to hex (will be more than 64 chars, so we truncate)
		apiKey := hex.EncodeToString(encrypted)
		if len(apiKey) > 64 {
			apiKey = apiKey[:64]
		}

		_, err = conn.Exec(ctx, `
			UPDATE users SET embapi_key = $1, updated_at = NOW() WHERE user_handle = '_system'
		`, apiKey)
		if err != nil {
			// Don't expose the key in error messages
			return fmt.Errorf("failed to update _system user API key in database")
		}

		fmt.Println("    Updated _system user with secure API key")
	}

	return nil
}
