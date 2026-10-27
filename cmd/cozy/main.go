// Command cozy is the cozy-mug tool: project scaffolding, migrations, settings
// helpers, and (later) code generation. It is stdlib-only.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "new":
		if err := runNew(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "migrate":
		if err := runMigrate(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "seed":
		if err := runSeed(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "console":
		if err := runConsole(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "dev":
		if err := runDev(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "version", "--version", "-v":
		if v := resolvedVersion(); v != "" {
			fmt.Printf("cozy %s\n", strings.TrimPrefix(v, "v"))
		} else {
			fmt.Println("cozy (devel)")
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: cozy <new|migrate|seed|console|dev|version> [args]")
	fmt.Fprintln(os.Stderr, "  cozy new <name> [--dir DIR] [--template example|minimal|auth|polls] [--db sqlite|postgres] [--git]")
	fmt.Fprintln(os.Stderr, "  cozy migrate [new <name> [source-dir]|up]  apply migrations (default: up)")
	fmt.Fprintln(os.Stderr, "  cozy seed                                 apply idempotent seed SQL from seeds/")
	fmt.Fprintln(os.Stderr, "  cozy console                              database + routes REPL")
	fmt.Fprintln(os.Stderr, "  cozy dev [--dir DIR] [--port PORT]    build/restart/refresh dev loop")
	fmt.Fprintln(os.Stderr, "  cozy version                           print the version")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "cozy:", err)
	os.Exit(1)
}
