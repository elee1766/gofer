package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/elee1766/gofer/src/config"
	"github.com/elee1766/gofer/src/storage"
)

// MigrateCmd manages database migrations
type MigrateCmd struct {
	Up     MigrateUpCmd     `cmd:"" help:"Run pending migrations"`
	Down   MigrateDownCmd   `cmd:"" help:"Rollback last migration"`
	Status MigrateStatusCmd `cmd:"" help:"Show migration status"`
}

// MigrateUpCmd runs pending migrations
type MigrateUpCmd struct {
	DBPath string `help:"Database path (defaults to config)"`
}

// Run executes the migrate up command
func (c *MigrateUpCmd) Run(ctx *kong.Context, cli *CLI) error {
	dbPath := c.DBPath
	if dbPath == "" {
		paths := config.GetDefaultStoragePaths()
		dbPath = paths.DatabasePath
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Database opened: %s (migrations handled automatically)\n", dbPath)
	return nil
}

// MigrateDownCmd rolls back the last migration
type MigrateDownCmd struct {
	DBPath string `help:"Database path (defaults to config)"`
}

// Run executes the migrate down command
func (c *MigrateDownCmd) Run(ctx *kong.Context, cli *CLI) error {
	return fmt.Errorf("migration rollback not supported in current storage implementation")
}

// MigrateStatusCmd shows migration status
type MigrateStatusCmd struct {
	DBPath string `help:"Database path (defaults to config)"`
}

// Run executes the migrate status command
func (c *MigrateStatusCmd) Run(ctx *kong.Context, cli *CLI) error {
	return fmt.Errorf("migration status not supported in current storage implementation")
}