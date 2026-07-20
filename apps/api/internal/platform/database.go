package platform

import (
	"context"

	"github.com/uptrace/bun"
)

type DatabaseClient interface {
	// Ping checks the health of the Postgres connection pool.
	Ping(ctx context.Context) error

	// Close drains the Bun and SQL connection pools.
	Close() error

	// DB returns Bun's unified IDB interface for running queries.
	DB() bun.IDB

	// RunInTx executes a function inside a database transaction.
	// Returns an error and rolls back if the function fails.
	RunInTx(ctx context.Context, fn func(ctx context.Context, tx bun.Tx) error) error
}
