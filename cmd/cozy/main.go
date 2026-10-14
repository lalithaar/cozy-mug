// Command cozy is the cozy-mug tool: project scaffolding, migrations, settings
// helpers, and (later) code generation. It is stdlib-only.
package main

import (
	"fmt"
	"os"
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
	case "version", "--version", "-v":
		fmt.Println("cozy 0.0.11")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: cozy <new|migrate<|start|>version> [args]")
	fmt.Fprintln(os.Stderr, "  cozy new <name> [--dir DIR]            scaffold a new app")
	fmt.Fprintln(os.Stderr, "  cozy migrate [new <name> [source-dir]|up]  apply migrations (default: up)")
	fmt.Fprintln(os.Stderr, "  cozy version                           print the version")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "cozy:", err)
	os.Exit(1)
}
