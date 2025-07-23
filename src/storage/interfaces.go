package storage

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
)

// Execer is an interface for executing SQL statements
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// ExecQuerier combines both Execer and sqlscan.Querier interfaces
// for operations that need both SELECT and INSERT/UPDATE/DELETE capabilities
type ExecQuerier interface {
	Execer
	sqlscan.Querier
}