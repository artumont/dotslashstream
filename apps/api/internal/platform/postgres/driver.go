package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/artumont/dotslashstream/internal/platform"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type BunPostgresDriver struct {
	client *bun.DB
}

var _ platform.DatabaseClient = (*BunPostgresDriver)(nil)

func New(dsn string) (*BunPostgresDriver, error) {
	conn := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(conn)

	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(25)
	sqldb.SetConnMaxLifetime(5 * time.Minute)

	bunClient := bun.NewDB(sqldb, pgdialect.New())

	ctx := context.Background()
	if err := RunMigrations(ctx, bunClient); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &BunPostgresDriver{client: bunClient}, nil
}

func (m *BunPostgresDriver) Ping(ctx context.Context) error {
	return m.client.PingContext(ctx)
}

func (m *BunPostgresDriver) Close() error {
	return m.client.Close()
}

func (m *BunPostgresDriver) DB() bun.IDB {
	return m.client
}

func (m *BunPostgresDriver) RunInTx(ctx context.Context, fn func(ctx context.Context, tx bun.Tx) error) error {
	return m.client.RunInTx(ctx, nil, fn)
}
