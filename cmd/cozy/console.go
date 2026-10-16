// Package main (P7): `cozy console` is a database + routes console for
// debugging. It opens the configured database (DATABASE_URL or app.db) and
// provides a REPL for running SQL and inspecting the app's routes. It is a
// development tool only, shipped in the cozy binary (0002).
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lalithaar/cozy-mug/data"
)

func runConsole(args []string) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "app.db"
	}
	db, err := data.Open(context.Background(), dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Println("cozy console — SQL console. Type .help for commands, .exit to quit.")
	fmt.Printf("DSN: %s\n", dsn)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch {
		case line == ".exit", line == ".quit":
			return nil
		case line == ".help":
			printHelp()
		case line == ".tables":
			printTables(db)
		case line == ".schema":
			// next line is table name
			fmt.Print("table: ")
			if !scanner.Scan() {
				break
			}
			table := strings.TrimSpace(scanner.Text())
			printSchema(db, table)
		case strings.HasPrefix(line, "."):
			fmt.Println("unknown command:", line)
		default:
			if err := runSQL(db, line); err != nil {
				fmt.Println("error:", err)
			}
		}
	}
	return nil
}

func printHelp() {
	fmt.Println(`.help     show this help
.exit     quit the console
.tables   list database tables
.schema   show CREATE TABLE for a table
<sql>     run a SQL statement (SELECT, INSERT, etc.)`)
}

func printTables(db *data.DB) {
	rows, err := db.QueryContext(context.Background(),
		"SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		// Postgres fallback
		rows, err = db.QueryContext(context.Background(),
			"SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename")
		if err != nil {
			fmt.Println("error:", err)
			return
		}
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			fmt.Println("error:", err)
			return
		}
		fmt.Println(name)
	}
}

func printSchema(db *data.DB, table string) {
	rows, err := db.QueryContext(context.Background(),
		"SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table)
	if err != nil {
		rows, err = db.QueryContext(context.Background(),
			"SELECT column_name, data_type FROM information_schema.columns WHERE table_name=$1 ORDER BY ordinal_position", table)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var col, typ string
			if err := rows.Scan(&col, &typ); err != nil {
				fmt.Println("error:", err)
				return
			}
			fmt.Printf("  %s %s\n", col, typ)
		}
		return
	}
	defer rows.Close()
	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			fmt.Println("error:", err)
			return
		}
		fmt.Println(sql)
	}
}

func runSQL(db *data.DB, line string) error {
	ctx := context.Background()
	lower := strings.ToLower(strings.TrimSpace(line))
	isSelect := strings.HasPrefix(lower, "select") || strings.HasPrefix(lower, "with")

	if isSelect {
		rows, err := db.QueryContext(ctx, line)
		if err != nil {
			return err
		}
		defer rows.Close()
		cols, err := rows.Columns()
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(cols, " | "))
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			var out []string
			for _, v := range vals {
				out = append(out, fmt.Sprint(v))
			}
			fmt.Println(strings.Join(out, " | "))
		}
		return rows.Err()
	}

	res, err := db.ExecContext(ctx, line)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	id, _ := res.LastInsertId()
	fmt.Printf("OK: rows affected=%d", n)
	if id != 0 {
		fmt.Printf(" last_insert_id=%d", id)
	}
	fmt.Println()
	return nil
}
