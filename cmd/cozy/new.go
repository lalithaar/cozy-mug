package main

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed testdata/project testdata/project-minimal testdata/project-auth testdata/project-polls
var templateFS embed.FS

func runNew(args []string) error {
	name := ""
	var dir string
	var skipTidy bool
	var doGit bool
	templateName := "example"
	templateSet := false
	moduleOverride := ""
	db := "sqlite" // default database
	dbSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-dir", "--dir":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			dir = args[i+1]
			i++
		case "--template", "-template", "-t":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			templateName = args[i+1]
			templateSet = true
			i++
		case "--module", "-module":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			moduleOverride = args[i+1]
			i++
		case "--db", "--database":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			db = args[i+1]
			dbSet = true
			i++
		case "--git":
			doGit = true
		case "--auth", "-auth":
			templateName = "auth"
			templateSet = true
		case "--skip-tidy", "-skip-tidy":
			skipTidy = true
		default:
			if name == "" {
				name = args[i]
			} else {
				return fmt.Errorf("unexpected argument %q", args[i])
			}
		}
	}
	validTemplates := map[string]bool{"example": true, "minimal": true, "auth": true, "polls": true}
	if !validTemplates[templateName] {
		return fmt.Errorf("unknown template %q (want example, minimal, auth, or polls)", templateName)
	}
	validDBs := map[string]bool{"sqlite": true, "postgres": true}
	if !validDBs[db] {
		return fmt.Errorf("unknown database %q (want sqlite or postgres)", db)
	}
	if name == "" {
		// Works both interactively and piped (e.g. echo name | cozy new).
		if isTerminal(os.Stdin) {
			fmt.Print("Project name: ")
		}
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			return fmt.Errorf("usage: cozy new <name> [--dir DIR]")
		}
		name = strings.TrimSpace(line)
		if name == "" {
			return fmt.Errorf("usage: cozy new <name> [--dir DIR]")
		}
	}

	if !templateSet && isTerminal(os.Stdin) {
		fmt.Print("Template (example, minimal, auth, polls) [example]: ")
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		if tname := strings.TrimSpace(line); tname != "" {
			templateName = tname
		}
	}

	if !dbSet && isTerminal(os.Stdin) {
		fmt.Printf("Database (sqlite, postgres) [%s]: ", db)
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		if d := strings.TrimSpace(line); d != "" {
			db = d
		}
	}

	if moduleOverride == "" && isTerminal(os.Stdin) {
		defaultModule := name
		if !strings.Contains(defaultModule, ".") && !strings.Contains(defaultModule, "/") {
			defaultModule = "example.com/" + defaultModule
		}
		fmt.Printf("Module path [%s]: ", defaultModule)
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		if m := strings.TrimSpace(line); m != "" {
			moduleOverride = m
		}
	}

	if !doGit && isTerminal(os.Stdin) {
		fmt.Print("Initialize git repository? (y/N): ")
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		if strings.EqualFold(strings.TrimSpace(line), "y") {
			doGit = true
		}
	}
	if !validTemplates[templateName] {
		return fmt.Errorf("unknown template %q (want example, minimal, auth, or polls)", templateName)
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
	if moduleOverride != "" {
		module = moduleOverride
	} else if !strings.Contains(module, ".") && !strings.Contains(module, "/") {
		// conventional Go module path; user can edit go.mod later
		module = "example.com/" + module
	}

	root := "testdata/project"
	if templateName == "minimal" {
		root = "testdata/project-minimal"
	} else if templateName == "auth" {
		root = "testdata/project-auth"
	} else if templateName == "polls" {
		root = "testdata/project-polls"
	}
	if err := copyTemplate(root, dir, module, db); err != nil {
		return err
	}
	frameworkVersion := resolvedVersion()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomodFile(module, frameworkVersion)), 0o644); err != nil {
		return err
	}
	// A tiny .gitignore so VCS hygiene works out of the box.
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("# Build artifacts\n*.exe\n*.test\n*.out\n\n# Local databases\n*.db\n*.db-shm\n*.db-wal\n*.sqlite\n*.sqlite3\n"), 0o644)

	fmt.Printf("created %s\n", dir)
	if frameworkVersion == "" {
		fmt.Println("cozy new: development build - the framework version is not pinned; `go mod tidy` resolves the latest release")
	}

	if !skipTidy {
		fmt.Println("fetching dependencies (go mod tidy)...")
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("cozy new: go mod tidy failed; run it manually:")
			fmt.Printf("  cd %s && go mod tidy\n", dir)
		}
	}

	if doGit {
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	fmt.Println("next:")
	fmt.Printf("  cd %s\n", dir)
	fmt.Printf("  go run .            # http://localhost:8080\n")
	if templateName == "example" {
		fmt.Printf("  cozy migrate new init .   # generate your first migration (already committed as 0001_init)\n")
	}
	fmt.Printf("  cozy dev            # build -> restart -> refresh\n")
	return nil
}

// gomodFile renders the scaffold's go.mod. The framework requirement pins the
// CLI's own version when it can be determined (release builds via -ldflags,
// module installs via build info). A development build knows no version, so the
// require line is omitted and `go mod tidy` resolves the latest release.
func gomodFile(module, frameworkVersion string) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "module %s\n\ngo 1.27\n", module)
	if frameworkVersion != "" {
		fmt.Fprintf(b, "\nrequire %s %s\n", modulePath, frameworkVersion)
	}
	return b.String()
}

// copyTemplate copies the template directory to dest, replacing __MODULE__ with
// the module path. When copying the polls template, only the chosen database
// dialect's migration directory is kept.
func copyTemplate(root, dest, module, dialect string) error {
	return fs.WalkDir(templateFS, root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, root)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		// For the polls template, only keep the chosen dialect's migrations.
		if strings.Contains(root, "project-polls") {
			relSlash := strings.ReplaceAll(rel, string(filepath.Separator), "/")
			if strings.Contains(relSlash, "migrations/") {
				parts := strings.Split(relSlash, "/")
				if len(parts) >= 2 && parts[0] == "migrations" && parts[1] != dialect {
					return nil // skip the non-chosen dialect
				}
			}
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

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
