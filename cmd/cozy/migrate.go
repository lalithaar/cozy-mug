package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lalithaar/cozy-mug/data"
)

type migrateFlags struct {
	db    string
	dir   string
	dsn   string
}

func defaultDSN() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return "app.db"
}

func runMigrate(args []string) error {
	// subcommand
	if len(args) > 0 && (args[0] == "new" || args[0] == "create") {
		return migrateNew(args[1:])
	}
	return migrateUp(args)
}

func migrateFlagsFrom(args []string) (migrateFlags, []string, error) {
	f := migrateFlags{db: defaultDSN(), dir: "migrations"}
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-db", "--db", "-database", "--database":
			if i+1 >= len(args) {
				return f, nil, fmt.Errorf("missing value for %s", args[i])
			}
			f.db = args[i+1]
			i++
		case "-dir", "--dir":
			if i+1 >= len(args) {
				return f, nil, fmt.Errorf("missing value for %s", args[i])
			}
			f.dir = args[i+1]
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	f.dsn = f.db
	return f, rest, nil
}

func dialectOf(dsn string) string {
	l := strings.ToLower(dsn)
	if strings.HasPrefix(l, "postgres://") || strings.HasPrefix(l, "postgresql://") {
		return "postgres"
	}
	return "sqlite"
}

func migrateUp(args []string) error {
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

	dialect := dialectOf(f.dsn)
	dir := filepath.Join(f.dir, dialect)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("migration directory not found: %s", dir)
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
		fmt.Println("no migration files found in", dir)
		return nil
	}

	schema := `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, "SELECT name FROM schema_migrations")
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

	insertSQL := "INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)"
	if dialect == "postgres" {
		insertSQL = "INSERT INTO schema_migrations (name, applied_at) VALUES ($1, $2)"
	}

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

func migrateNew(args []string) error {
	f, rest, err := migrateFlagsFrom(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return fmt.Errorf("usage: cozy migrate new <name>")
	}
	name := sanitizeName(rest[0])
	if name == "" {
		return fmt.Errorf("invalid migration name")
	}

	// number
	n := nextNumber(f.dir)
	for _, d := range []string{"sqlite", "postgres"} {
		dir := filepath.Join(f.dir, d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		file := filepath.Join(dir, fmt.Sprintf("%04d_%s.sql", n, name))
		if _, err := os.Stat(file); err == nil {
			continue
		}
		stub := "-- Write your reviewable SQL here.\n-- One file applies on both? No: this file is the " + d + " variant.\n"
		if err := os.WriteFile(file, []byte(stub), 0o644); err != nil {
			return err
		}
		fmt.Println("created", file)
	}
	return nil
}

func nextNumber(dir string) int {
	max := 0
	for _, d := range []string{"sqlite", "postgres"} {
		entries, err := os.ReadDir(filepath.Join(dir, d))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
				continue
			}
			parts := strings.SplitN(e.Name(), "_", 2)
			if len(parts) == 0 {
				continue
			}
			if n, err := strconv.Atoi(parts[0]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == '-' || r == ' ' || r == '_' {
			b.WriteRune('_')
		}
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return strings.Trim(out, "_")
}

func timeNow() string { return time.Now().UTC().Format(time.RFC3339) }

