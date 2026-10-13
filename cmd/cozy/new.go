package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed testdata/project
var templateFS embed.FS

func runNew(args []string) error {
	name := ""
	var dir string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-dir", "--dir":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			dir = args[i+1]
			i++
		default:
			if name == "" {
				name = args[i]
			} else {
				return fmt.Errorf("unexpected argument %q", args[i])
			}
		}
	}
	if name == "" {
		return fmt.Errorf("usage: cozy new <name> [--dir DIR]")
	}
	if dir == "" {
		dir = sanitizeName(name)
		if dir == "" {
			dir = "app"
		}
	}
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	module := name
	if !strings.Contains(module, ".") && !strings.Contains(module, "/") {
		// conventional Go module path; user can edit go.mod later
		module = "example.com/" + module
	}

	root := "testdata/project"
	err := copyTemplate(root, dir, module)
	if err != nil {
		return err
	}
	gomod := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/lalithaar/cozy-mug v0.0.5\n", module)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return err
	}
	fmt.Printf("created %s\n", dir)
	fmt.Println("next:")
	fmt.Printf("  cd %s\n", dir)
	fmt.Println("  go mod tidy")
	fmt.Println("  go run .")
	return nil
}

func copyTemplate(root, dest, module string) error {
	return fs.WalkDir(templateFS, root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, root)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := templateFS.ReadFile(p)
		if err != nil {
			return err
		}
		content := strings.ReplaceAll(string(data), "__MODULE__", module)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(content), 0o644)
	})
}
