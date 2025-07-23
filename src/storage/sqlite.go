package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/sqlite/001_initial_schema.sql
var initialSchema string

//go:embed migrations/sqlite/002_sessions_json_array.sql
var sessionsJSONArray string

//go:embed migrations/sqlite/003_add_tool_calls_to_messages.sql
var addToolCallsToMessages string

type DB struct {
	path string
	db   *sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	
	store := &DB{path: path, db: db}
	
	// Run migrations
	if err := store.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	
	return store, nil
}

func (d *DB) DB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

// runMigrations runs database migrations
func (d *DB) runMigrations() error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	
	if _, err := d.db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	// Check which migrations have been applied
	var appliedVersions []int
	rows, err := d.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		appliedVersions = append(appliedVersions, version)
	}
	
	// Define migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{1, extractUpMigration(initialSchema)},
		{2, extractUpMigration(sessionsJSONArray)},
		{3, extractUpMigration(addToolCallsToMessages)},
	}
	
	// Apply pending migrations
	for _, migration := range migrations {
		if contains(appliedVersions, migration.version) {
			continue
		}
		
		// Begin transaction
		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		
		// Execute migration
		if _, err := tx.Exec(migration.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.version, err)
		}
		
		// Record migration
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.version, err)
		}
		
		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.version, err)
		}
	}
	
	return nil
}

// extractUpMigration extracts the UP migration from goose format
func extractUpMigration(content string) string {
	lines := strings.Split(content, "\n")
	var upMigration []string
	inUp := false
	inStatement := false
	
	for _, line := range lines {
		if strings.Contains(line, "-- +goose Up") {
			inUp = true
			continue
		}
		if strings.Contains(line, "-- +goose Down") {
			break
		}
		if strings.Contains(line, "-- +goose StatementBegin") {
			inStatement = true
			continue
		}
		if strings.Contains(line, "-- +goose StatementEnd") {
			inStatement = false
			continue
		}
		if inUp && inStatement {
			upMigration = append(upMigration, line)
		}
	}
	
	return strings.Join(upMigration, "\n")
}

// contains checks if a slice contains a value
func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
