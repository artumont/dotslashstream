package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type BunPostgresManager struct {
	client *bun.DB
}

func NewBunPostgresManager(dsn string) (*BunPostgresManager, error) {
	conn := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(conn)

	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(25)
	sqldb.SetConnMaxLifetime(5 * time.Minute)

	bunClient := bun.NewDB(sqldb, pgdialect.New())

	return &BunPostgresManager{client: bunClient}, nil
}

func (m *BunPostgresManager) Ping(ctx context.Context) error {
	return m.client.PingContext(ctx)
}

func (m *BunPostgresManager) Close() error {
	return m.client.Close()
}

func (m *BunPostgresManager) DB() bun.IDB {
	return m.client
}

func (m *BunPostgresManager) RunInTx(ctx context.Context, fn func(ctx context.Context, tx bun.Tx) error) error {
	return m.client.RunInTx(ctx, nil, fn)
}
