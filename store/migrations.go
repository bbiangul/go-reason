package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// migration represents a single schema migration.
type migration struct {
	version     int
	description string
	apply       func(tx *sql.Tx) error
}

// migrations is the ordered list of all schema migrations.
// New migrations are appended at the end; never modify existing entries.
var migrations = []migration{
	{
		version:     1,
		description: "initial schema (applied via schemaSQL)",
		apply:       func(tx *sql.Tx) error { return nil }, // base schema applied separately
	},
	{
		version:     2,
		description: "add token tracking to query_log",
		apply: func(tx *sql.Tx) error {
			// These columns were added in the base schema, so they may already
			// exist. Use a safe idempotent approach.
			for _, col := range []string{
				"ALTER TABLE query_log ADD COLUMN prompt_tokens INTEGER DEFAULT 0",
				"ALTER TABLE query_log ADD COLUMN completion_tokens INTEGER DEFAULT 0",
				"ALTER TABLE query_log ADD COLUMN total_tokens INTEGER DEFAULT 0",
			} {
				if _, err := tx.Exec(col); err != nil {
					// Column likely already exists — that's fine.
					slog.Debug("migration 2: column may already exist", "sql", col, "error", err)
				}
			}
			return nil
		},
	},
}

// Migrate runs all pending schema migrations.
func (s *Store) Migrate(ctx context.Context) error {
	// Ensure the schema_version table exists.
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			description TEXT,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("creating schema_version table: %w", err)
	}

	// Get current version.
	var current int
	row := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}

		slog.Info("applying migration", "version", m.version, "description", m.description)

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.version, err)
		}

		if err := m.apply(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", m.version, err)
		}

		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_version (version, description) VALUES (?, ?)",
			m.version, m.description); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", m.version, err)
		}
	}

	return nil
}
