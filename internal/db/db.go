// Package db buka koneksi SQLite dan menjalankan migration bawaan (embed).
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open bikin folder induk file DB kalau belum ada, buka koneksi, pasang
// pragma yang wajib buat sqlite di aplikasi konkuren (WAL + foreign_keys),
// lalu jalanin migration yang belum diterapkan.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}
	dsn := path
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Pool dibatasi kecil (bukan 1): WAL + busy_timeout di bawah ini yang
	// nanganin serialisasi writer di level SQLite. Kalau dipaksa 1 koneksi,
	// query nested dari goroutine yang sama (Rows belum ditutup lalu query
	// lagi) bisa saling tunggu connection yang sama → deadlock.
	database.SetMaxOpenConns(4)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	} {
		if _, err := database.Exec(pragma); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", pragma, err)
		}
	}

	if err := migrate(database); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return database, nil
}

func migrate(database *sql.DB) error {
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var already int
		err := database.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE filename = ?`, name).Scan(&already)
		if err != nil {
			return err
		}
		if already > 0 {
			continue
		}
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (filename) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
