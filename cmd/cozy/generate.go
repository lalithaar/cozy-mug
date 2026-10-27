// Package main (P1): `cozy generate` emits model structs and reviewable
// migration SQL from column specs, reusing the model-discovery/DDL rendering
// in gen.go. It is the first half of scaffold generation; `cozy generate
// scaffold` adds handler/view stubs.
//
// Usage:
//
//	cozy generate model Person name:string age:int email:string
//	cozy generate scaffold Document person_id:int doc_type:string file_key:string
package main

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

// runGenerate handles `cozy generate <subcommand>`.
func runGenerate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cozy generate <model|scaffold> <name> [col:type ...]")
	}
	switch args[0] {
	case "model":
		return runGenerateModel(args[1:])
	case "scaffold":
		return runGenerateScaffold(args[1:])
	default:
		return fmt.Errorf("unknown generate subcommand %q (want model|scaffold)", args[0])
	}
}

// colSpecFrom parses a single "name:type" column spec (or the older
// "name type" form) into a colSpec reused by the migration renderer in gen.go.
// type maps to a Go scalar; "auto" marks the auto-increment primary key
// (e.g. "id:int auto").
func colSpecFrom(spec string) (colSpec, error) {
	parts := strings.Fields(spec)
	if len(parts) == 0 {
		return colSpec{}, fmt.Errorf("invalid column spec %q (want name:type [flags])", spec)
	}
	var name, typeName string
	rest := parts[1:]
	if i := strings.IndexByte(parts[0], ':'); i > 0 {
		name, typeName = parts[0][:i], parts[0][i+1:]
	} else if len(parts) >= 2 {
		name, typeName = parts[0], parts[1]
		rest = parts[2:]
	}
	if name == "" || typeName == "" {
		return colSpec{}, fmt.Errorf("invalid column spec %q (want name:type [flags])", spec)
	}
	flags := map[string]bool{}
	for _, p := range rest {
		flags[p] = true
	}

	expr, ok := goTypeExpr(typeName)
	if !ok {
		return colSpec{}, fmt.Errorf("unsupported column type %q (want string|int|text|bool|...)", typeName)
	}

	return colSpec{
		name:   name,
		expr:   expr,
		auto:   flags["auto"],
		index:  flags["index"],
		unique: flags["unique"],
	}, nil
}

// goTypeExpr returns an ast.Expr for a scalar type name usable by scalarSQL.
func goTypeExpr(typeName string) (ast.Expr, bool) {
	switch typeName {
	case "string", "text":
		return &ast.Ident{Name: "string"}, true
	case "int", "integer":
		return &ast.Ident{Name: "int"}, true
	case "int64":
		return &ast.Ident{Name: "int64"}, true
	case "bool":
		return &ast.Ident{Name: "bool"}, true
	case "float", "float64":
		return &ast.Ident{Name: "float64"}, true
	case "time", "time.Time":
		return &ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}}, true
	default:
		return nil, false
	}
}

// runGenerateModel generates a model Go file and per-dialect migration SQL.
func runGenerateModel(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cozy generate model <Name> <col:type ...>")
	}
	name := args[0]
	table := strings.ToLower(name)
	if !strings.ContainsAny(table, "_") {
		// crude pluralization: append "s" unless already ends in s
		if !strings.HasSuffix(table, "s") {
			table += "s"
		}
	}

	var cols []colSpec
	hasID := false
	for _, spec := range args[1:] {
		cs, err := colSpecFrom(spec)
		if err != nil {
			return err
		}
		if cs.name == "id" && cs.auto {
			hasID = true
		}
		cols = append(cols, cs)
	}
	if !hasID {
		// prepend an auto id primary key
		idExpr, _ := goTypeExpr("int")
		cols = append([]colSpec{{name: "id", expr: idExpr, auto: true}}, cols...)
	}
	// always add maintained timestamps + version
	tsExpr, _ := goTypeExpr("time")
	cols = append(cols,
		colSpec{name: "created_at", expr: tsExpr},
		colSpec{name: "updated_at", expr: tsExpr},
		colSpec{name: "version", expr: &ast.Ident{Name: "int64"}},
	)

	specs := []modelSpec{{table: table, cols: cols}}

	// write migration files
	n := nextNumber("migrations")
	for _, d := range []string{"sqlite", "postgres"} {
		dir := filepath.Join("migrations", d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		file := filepath.Join(dir, fmt.Sprintf("%04d_create_%s.sql", n, table))
		content := renderMigration(specs, d) + "\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Println("created", file)
	}

	// write model Go file under models/
	modelDir := "models"
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return err
	}
	modelFile := filepath.Join(modelDir, strings.ToLower(name)+".go")
	if _, err := os.Stat(modelFile); err == nil {
		fmt.Println("skip (exists):", modelFile)
	} else {
		if err := writeModelFile(modelFile, name, table, cols); err != nil {
			return err
		}
		fmt.Println("created", modelFile)
	}
	fmt.Println()
	fmt.Println("Next: cozy migrate up")
	return nil
}

func runGenerateScaffold(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cozy generate scaffold <Name> <col:type ...>")
	}
	if err := runGenerateModel(args); err != nil {
		return err
	}
	name := args[0]
	// Scaffold stubs: handler + views. Minimal, meant to be edited.
	if err := writeHandlerStub(name); err != nil {
		return err
	}
	if err := writeViewStubs(name); err != nil {
		return err
	}
	fmt.Println("created scaffold stubs for", name)
	fmt.Println("Edit handlers/<name>.go and views/<name>/, then register routes.")
	return nil
}

func writeModelFile(path, name, table string, cols []colSpec) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("// Code generated by cozy generate. DO NOT EDIT.\npackage handlers\n\nimport (\n\t\"time\"\n\n\t\"github.com/lalithaar/cozy-mug/data\"\n)\n\n// %s is a generated model. Edit as needed.\ntype %s struct {\n\tdata.Model `db:\"%s\"`\n", name, name, table))
	// column tokens
	b.WriteString("\tID int64 `db:\"id auto\"`\n")
	b.WriteString("\tCreatedAt time.Time `db:\"created_at\"`\n")
	b.WriteString("\tUpdatedAt time.Time `db:\"updated_at\"`\n")
	b.WriteString("\tVersion int64 `db:\"version\"`\n")
	// user columns (skip id/timestamps/version which are written above)
	skip := map[string]bool{"id": true, "created_at": true, "updated_at": true, "version": true}
	for _, c := range cols {
		if skip[c.name] {
			continue
		}
		goType := goTypeString(c.expr)
		b.WriteString(fmt.Sprintf("\t%s %s `db:\"%s\"`\n", goFieldName(c.name), goType, c.name))
	}
	b.WriteString("}\n")
	b.WriteString(fmt.Sprintf("var %sCols = struct {\n", name))
	tokenNames := []string{}
	for _, c := range cols {
		if skip[c.name] {
			continue
		}
		tokenNames = append(tokenNames, c.name)
	}
	// we can't easily build the token struct generically; write a placeholder
	b.WriteString(fmt.Sprintf("\t// ID, CreatedAt, UpdatedAt, Version are maintained by data.Model\n\t// Add tokens for user columns via data.C[%s](\"col\")\n}\n", name))
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func goFieldName(col string) string {
	runes := []rune(col)
	if len(runes) == 0 {
		return ""
	}
	runes[0] = rune(toUpper(runes[0]))
	return string(runes)
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}

func goTypeString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	default:
		return "string"
	}
}

func writeHandlerStub(name string) error {
	lower := strings.ToLower(name)
	path := filepath.Join("handlers", lower+"_generated.go")
	var b strings.Builder
	b.WriteString("// Code generated by cozy generate scaffold. Edit as needed.\n")
	b.WriteString("package handlers\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"net/http\"\n")
	b.WriteString("\t\"time\"\n\n")
	b.WriteString("\t\"github.com/lalithaar/cozy-mug/core\"\n")
	b.WriteString("\t\"github.com/lalithaar/cozy-mug/data\"\n")
	b.WriteString("\t\"github.com/lalithaar/cozy-mug/router\"\n")
	b.WriteString(")\n\n")
	b.WriteString(fmt.Sprintf("// %s is a generated model placeholder; define it in models/ or handlers/.\n", name))
	b.WriteString(fmt.Sprintf("type %s struct {\n", name))
	b.WriteString("\tdata.Model `db:\"" + lower + "`\n")
	b.WriteString("\tID         int64     `db:\"id auto\"`\n")
	b.WriteString("\tCreatedAt  time.Time `db:\"created_at\"`\n")
	b.WriteString("\tUpdatedAt  time.Time `db:\"updated_at\"`\n")
	b.WriteString("\tVersion    int64     `db:\"version\"`\n")
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("// Register%sRoutes is a stub; wire it in your Store.Register.\n", name))
	b.WriteString(fmt.Sprintf("func Register%sRoutes(r *router.Router) {\n", name))
	b.WriteString(fmt.Sprintf("\tr.Get(\"/%s\", func(c *core.Context) core.Response {\n", lower))
	b.WriteString("\t\treturn core.String(http.StatusOK, \"not implemented\")\n")
	b.WriteString("\t}).As(\"" + lower + ".index\")\n")
	b.WriteString("}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeViewStubs(name string) error {
	lower := strings.ToLower(name)
	dir := filepath.Join("views", lower)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err == nil {
		return nil
	}
	content := fmt.Sprintf(`{{define "title"}}%s{{end}}
{{define "content"}}
<h1>%s</h1>
<p>Scaffold stub — edit views/%s/index.html</p>
{{end}}
`, name, name, lower)
	return os.WriteFile(index, []byte(content), 0o644)
}
