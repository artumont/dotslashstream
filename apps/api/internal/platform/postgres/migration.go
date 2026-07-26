package postgres

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/uptrace/bun"
)

//go:embed migrations/*
var migrationFS embed.FS

const migrationsRoot = "internal/platform/postgres/migrations"

// ListAllMigrations returns all migrations from the filesystem, no DB needed.
func ListAllMigrations() ([]MigrationInfo, error) {
	dirs, err := discoverMigrations(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}

	var result []MigrationInfo
	for _, d := range dirs {
		result = append(result, MigrationInfo{
			Num:         d.Num,
			Name:        d.Meta.Name,
			Description: d.Meta.Description,
			Author:      d.Meta.Author,
			Safe:        d.Meta.Safe,
		})
	}
	return result, nil
}

// EnsureMigrationsTable creates the migration tracking table if absent.
func EnsureMigrationsTable(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id         BIGSERIAL PRIMARY KEY,
			name       VARCHAR NOT NULL UNIQUE,
			author     VARCHAR NOT NULL DEFAULT 'unknown',
			applied_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

// appliedSet returns the set of already-applied migration names.
func appliedSet(ctx context.Context, db *bun.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		set[name] = true
	}
	return set, rows.Err()
}

// discoverMigrations reads migration folders from the given FS root.
func discoverMigrations(fsys fs.FS, root string) ([]migrationEntry, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", root, err)
	}

	var dirs []migrationEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(root, entry.Name())
		metaPath := filepath.Join(dirPath, "metadata.json")

		if _, err := fs.Stat(fsys, metaPath); err != nil {
			continue
		}

		data, err := fs.ReadFile(fsys, metaPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", metaPath, err)
		}

		var meta MigrationMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parse %s: %w", metaPath, err)
		}

		num := parseMigrationNum(entry.Name())
		dirs = append(dirs, migrationEntry{
			Path: dirPath,
			Meta: meta,
			Num:  num,
		})
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Num < dirs[j].Num
	})

	return dirs, nil
}

// parseMigrationNum extracts the leading number from a folder name.
func parseMigrationNum(name string) int {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0])
	return n
}

// RunMigrations applies pending migrations.
// safe: true  → auto-run on startup
// safe: false → only run when MIGRATE_UNSAFE=true
func RunMigrations(ctx context.Context, db *bun.DB) error {
	if err := EnsureMigrationsTable(ctx, db); err != nil {
		return err
	}

	applied, err := appliedSet(ctx, db)
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}

	dirs, err := discoverMigrations(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("discover migrations: %w", err)
	}

	unsafeEnabled := os.Getenv("MIGRATE_UNSAFE") == "true"

	for _, d := range dirs {
		if applied[d.Meta.Name] {
			continue
		}

		if !d.Meta.Safe && !unsafeEnabled {
			log.Printf("  Skipped %s (unsafe; set MIGRATE_UNSAFE=true to apply)\n", d.Meta.Name)
			continue
		}

		sqlContent, err := readSQL(migrationFS, d.Path, "up.sql")
		if err != nil {
			return err
		}

		if _, err := db.ExecContext(ctx, sqlContent); err != nil {
			return fmt.Errorf("apply %s: %w", d.Meta.Name, err)
		}

		if _, err := db.ExecContext(ctx,
			"INSERT INTO schema_migrations (name, author) VALUES (?, ?)", d.Meta.Name, d.Meta.Author,
		); err != nil {
			return fmt.Errorf("record %s: %w", d.Meta.Name, err)
		}

		log.Printf("  Migrated %s\n", d.Meta.Name)
	}

	return nil
}

// ListMigrations returns all migrations with applied status.
func ListMigrations(ctx context.Context, db *bun.DB) ([]MigrationRow, error) {
	applied, err := appliedSet(ctx, db)
	if err != nil {
		return nil, err
	}

	dirs, err := discoverMigrations(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}

	var result []MigrationRow
	for _, d := range dirs {
		result = append(result, MigrationRow{
			migrationEntry: d,
			Applied:        applied[d.Meta.Name],
		})
	}

	return result, nil
}

// RollbackMigrations rolls back the last applied migration.
func RollbackMigrations(ctx context.Context, db *bun.DB) error {
	if err := EnsureMigrationsTable(ctx, db); err != nil {
		return err
	}

	var name string
	err := db.QueryRowContext(ctx,
		"SELECT name FROM schema_migrations ORDER BY id DESC LIMIT 1",
	).Scan(&name)
	if err == sql.ErrNoRows {
		fmt.Println("Nothing to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("find last migration: %w", err)
	}

	dirs, err := discoverMigrations(migrationFS, "migrations")
	if err != nil {
		return err
	}

	for _, d := range dirs {
		if d.Meta.Name != name {
			continue
		}

		downSQL, err := readSQL(migrationFS, d.Path, "down.sql")
		if err != nil {
			return fmt.Errorf("read down.sql for %s: %w", name, err)
		}

		if _, err := db.ExecContext(ctx, downSQL); err != nil {
			return fmt.Errorf("rollback %s: %w", name, err)
		}

		if _, err := db.ExecContext(ctx,
			"DELETE FROM schema_migrations WHERE name = ?", name,
		); err != nil {
			return fmt.Errorf("remove %s: %w", name, err)
		}

		fmt.Printf("  ✓ Rolled back %s\n", name)
		return nil
	}

	return fmt.Errorf("migration %s not found in filesystem", name)
}

// NextMigrationNum returns the next migration sequence number.
func NextMigrationNum() (int, error) {
	dirs, err := discoverMigrations(migrationFS, "migrations")
	if err != nil {
		return 1, nil
	}

	maxNum := 0
	for _, d := range dirs {
		if d.Num > maxNum {
			maxNum = d.Num
		}
	}
	return maxNum + 1, nil
}

// CreateMigration creates a new migration folder with metadata.json, up.sql, down.sql.
func CreateMigration(name, description string, safe bool) (string, error) {
	num, err := NextMigrationNum()
	if err != nil {
		return "", err
	}

	author := gitAuthor()
	folderName := fmt.Sprintf("%06d_%s", num, name)
	dir := filepath.Join(migrationsRoot, folderName)

	meta := MigrationMeta{
		Name:        name,
		Author:      author,
		Description: description,
		Safe:        safe,
	}

	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), metaJSON, 0o644); err != nil {
		return "", fmt.Errorf("write metadata: %w", err)
	}

	upContent := fmt.Sprintf("-- %s\n\n", description)
	if err := os.WriteFile(filepath.Join(dir, "up.sql"), []byte(upContent), 0o644); err != nil {
		return "", fmt.Errorf("write up.sql: %w", err)
	}

	downContent := fmt.Sprintf("-- Rollback: %s\n\n", description)
	if err := os.WriteFile(filepath.Join(dir, "down.sql"), []byte(downContent), 0o644); err != nil {
		return "", fmt.Errorf("write down.sql: %w", err)
	}

	return dir, nil
}
