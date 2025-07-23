# Database Migrations

This directory contains database migrations managed by [Goose](https://github.com/pressly/goose).

## Directory Structure

- `sqlite/` - SQLite-specific migration files

## Migration Files

Migration files follow the Goose naming convention:
- `001_initial_schema.sql` - Initial database schema
- `002_feature_name.sql` - Future migrations

Each migration file contains:
- `-- +goose Up` section for applying the migration
- `-- +goose Down` section for rolling back the migration

## Usage

### Apply Migrations
```bash
# Run all pending migrations
gofer migrate up

# Check migration status
gofer migrate status
```

### Rollback Migrations
```bash
# Rollback the last migration
gofer migrate down
```

### Custom Database Path
```bash
# Use custom database path
gofer migrate up --db-path /path/to/custom.db
gofer migrate status --db-path /path/to/custom.db
```

## Creating New Migrations

1. Create a new migration file in `sqlite/` with the next sequence number:
   ```
   002_add_new_feature.sql
   ```

2. Add the SQL for both up and down migrations:
   ```sql
   -- +goose Up
   -- +goose StatementBegin
   CREATE TABLE new_table (...);
   -- +goose StatementEnd

   -- +goose Down
   -- +goose StatementBegin
   DROP TABLE IF EXISTS new_table;
   -- +goose StatementEnd
   ```

3. Migrations are automatically embedded in the binary and will be applied when the application starts.

## Features

- **Embedded Migrations**: Migration files are embedded in the binary
- **Automatic Application**: Migrations run automatically on startup
- **Version Tracking**: Goose tracks applied migrations in `goose_db_version` table
- **Rollback Support**: Safe rollback of migrations
- **Transaction Safety**: Each migration runs in a transaction
- **Schema Validation**: Foreign key constraints and data integrity

## Best Practices

1. **Always include a rollback**: Every migration should have a corresponding down migration
2. **Test migrations**: Test both up and down migrations before committing
3. **Backward compatibility**: Ensure migrations don't break existing data
4. **Atomic changes**: Keep each migration focused on a single change
5. **Data migrations**: Use separate migrations for schema and data changes