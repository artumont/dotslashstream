package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/artumont/dotslashstream/internal/platform/postgres"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "list":
		listCmd()
	case "create":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: migrate create <name> <description> [--unsafe]")
			os.Exit(1)
		}
		createCmd(os.Args[2:])
	case "up":
		withDB(up)
	case "down":
		withDB(down)
	case "status":
		withDB(status)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: migrate <command>

commands:
  list                           List all migrations with metadata
  create <name> <desc>           Create a new safe migration
  create <name> <desc> --unsafe  Create an unsafe migration
  up                             Run pending migrations
  down                           Rollback the last applied migration
  status                         Show applied migrations`)
}

// dbDSN reads DB_DSN from env. Only called for DB commands.
func dbDSN() string {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN not set. Export it or add to .env")
	}
	return dsn
}

func connectDB() *bun.DB {
	conn := pgdriver.NewConnector(pgdriver.WithDSN(dbDSN()))
	sqldb := sql.OpenDB(conn)
	return bun.NewDB(sqldb, pgdialect.New())
}

func withDB(fn func(ctx context.Context, db *bun.DB) error) {
	godotenv.Load() // load .env if present

	ctx := context.Background()
	db := connectDB()
	defer db.Close()

	if err := postgres.EnsureMigrationsTable(ctx, db); err != nil {
		log.Fatal(err)
	}

	if err := fn(ctx, db); err != nil {
		log.Fatal(err)
	}
}

// ── Filesystem-only commands (no DB) ───────────────────────────────────────

func listCmd() {
	rows, err := postgres.ListAllMigrations()
	if err != nil {
		log.Fatal(err)
	}

	if len(rows) == 0 {
		fmt.Println("no migrations found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tNAME\tDESCRIPTION\tAUTHOR\tSAFE")
	for _, r := range rows {
		safe := "no"
		if r.Safe {
			safe = "yes"
		}
		fmt.Fprintf(w, "%06d\t%s\t%s\t%s\t%s\n",
			r.Num, r.Name, r.Description, r.Author, safe)
	}
	w.Flush()
}

func createCmd(args []string) {
	name := args[0]
	description := args[1]
	safe := true

	for _, a := range args[2:] {
		if a == "--unsafe" {
			safe = false
		}
	}

	dir, err := postgres.CreateMigration(name, description, safe)
	if err != nil {
		log.Fatalf("create migration: %v", err)
	}

	fmt.Printf("  created %s\n", dir)
	fmt.Printf("  edit:   %s/up.sql\n", dir)
	fmt.Printf("  undo:   %s/down.sql\n", dir)
}

// ── DB commands ────────────────────────────────────────────────────────────

func up(ctx context.Context, db *bun.DB) error {
	return postgres.RunMigrations(ctx, db)
}

func down(ctx context.Context, db *bun.DB) error {
	return postgres.RollbackMigrations(ctx, db)
}

func status(ctx context.Context, db *bun.DB) error {
	rows, err := postgres.ListMigrations(ctx, db)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Println("no migrations found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tNAME\tDESCRIPTION\tAUTHOR\tSAFE")
	for _, r := range rows {
		status := "pending"
		if r.Applied {
			status = "applied"
		}
		safe := "no"
		if r.Meta.Safe {
			safe = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			status, r.Meta.Name, r.Meta.Description, r.Meta.Author, safe)
	}
	w.Flush()
	return nil
}
