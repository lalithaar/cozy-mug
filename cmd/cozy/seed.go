// Package main (P5): `cozy seed` applies idempotent seed SQL files from the
// `seeds/` directory, tracked in a `seed_migrations` table so re-running is a
// no-op for already-applied files. Seed SQL should be idempotent by design
// (e.g. INSERT ... WHERE NOT EXISTS).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lalithaar/cozy-mug/data"
)

func runSeed(args []string) error {
	f, rest, err := migrateFlagsFrom(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected arguments: %v", rest)
	}

	ctx := context.Background()
	db, err := data.Open(ctx, f.dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	dir := filepath.Join(f.dir, "seeds")
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("seed directory not found: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Println("no seed files found in", dir)
		return nil
	}

	schema := `CREATE TABLE IF NOT EXISTS seed_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create seed_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, "SELECT name FROM seed_migrations")
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		applied[name] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	insertSQL := "INSERT INTO seed_migrations (name, applied_at) VALUES (?, ?)"

	for _, name := range files {
		if applied[name] {
			fmt.Println("skip", name)
			continue
		}
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sqlText := string(body)
		err = db.Tx(ctx, func(tx *data.Tx) error {
			if strings.TrimSpace(sqlText) != "" {
				if _, err := tx.ExecContext(ctx, sqlText); err != nil {
					return fmt.Errorf("exec %s: %w", name, err)
				}
			}
			if _, err := tx.ExecContext(ctx, insertSQL, name, timeNow()); err != nil {
				return fmt.Errorf("record %s: %w", name, err)
			}
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Println("applied", name)
	}
	return nil
}
