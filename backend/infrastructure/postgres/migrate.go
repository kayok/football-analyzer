package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *Store) Migrate(ctx context.Context, dir, command string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(7102401)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); err != nil {
		return err
	}
	var version int
	if err = tx.QueryRow(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version); err != nil {
		return err
	}
	switch command {
	case "status":
		fmt.Printf("Migration version: %d\n", version)
	case "up":
		files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
		if err != nil {
			return err
		}
		for _, file := range files {
			v, err := strconv.Atoi(strings.Split(filepath.Base(file), "_")[0])
			if err != nil {
				return err
			}
			if v <= version {
				continue
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, string(raw)); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", v); err != nil {
				return err
			}
			fmt.Printf("Applied migration %d\n", v)
		}
	case "down":
		if version == 0 {
			break
		}
		files, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%03d_*.down.sql", version)))
		if err != nil {
			return err
		}
		if len(files) != 1 {
			return fmt.Errorf("missing down migration %d", version)
		}
		raw, err := os.ReadFile(files[0])
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(raw)); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version=$1", version); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown migration command %q", command)
	}
	return tx.Commit(ctx)
}
