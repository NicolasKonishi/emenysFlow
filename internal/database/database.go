package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	migrationsPath := migrationDirectory()
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if err := applySQLFile(ctx, db, "schema_migrations", filepath.Join(migrationsPath, entry.Name()), entry.Name()); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// ApplyPrivateSeeds loads operational catalog data kept outside version control.
// Called from cmd/server after migrations; tests use Migrate only.
func ApplyPrivateSeeds(ctx context.Context, db *sql.DB) error {
	privatePath := privateSeedDirectory()
	entries, err := os.ReadDir(privatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read private seeds: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if err := applySQLFile(ctx, db, "private_seed_migrations", filepath.Join(privatePath, entry.Name()), entry.Name()); err != nil {
			return fmt.Errorf("private seed %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func applySQLFile(ctx context.Context, db *sql.DB, ledgerTable, path, version string) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		version TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`, ledgerTable)); err != nil {
		return fmt.Errorf("create ledger: %w", err)
	}

	var applied int
	err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE version = ?", ledgerTable), version).Scan(&applied)
	if err != nil {
		return fmt.Errorf("check version: %w", err)
	}
	if applied > 0 {
		return nil
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if _, err = tx.ExecContext(ctx, string(body)); err != nil {
		tx.Rollback()
		return fmt.Errorf("execute SQL: %w", err)
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(version, applied_at) VALUES(?, ?)", ledgerTable), version, time.Now().UTC().Format(time.RFC3339)); err != nil {
		tx.Rollback()
		return fmt.Errorf("record version: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func privateSeedDirectory() string {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "database", "seeds", "private")
	}
	return filepath.Join(filepath.Dir(sourceFile), "seeds", "private")
}

func migrationDirectory() string {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "database", "migrations")
	}
	return filepath.Join(filepath.Dir(sourceFile), "migrations")
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
