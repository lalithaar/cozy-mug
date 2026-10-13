package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// modelSpec is a parsed model declaration discovered from Go source.
type modelSpec struct {
	table string
	cols  []colSpec
}

type colSpec struct {
	name   string
	expr   ast.Expr
	auto   bool
	index  bool
	unique bool
}

// findModels walks srcDir and returns every struct embedding data.Model, with
// its columns parsed from db tags. Source that fails to parse is skipped, so
// the generator degrades (to a stub) rather than aborting a whole run.
func findModels(srcDir string) ([]modelSpec, error) {
	var specs []modelSpec
	seen := map[string]bool{}
	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// skip vendor/module cache debris
			if d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			gd, ok := n.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				return true
			}
			for _, s := range gd.Specs {
				ts, ok := s.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				table, isModel := modelTable(st)
				if !isModel || table == "" {
					continue
				}
				if seen[table] {
					continue
				}
				seen[table] = true
				var cols []colSpec
				collectColumns(st, &cols)
				specs = append(specs, modelSpec{table: table, cols: cols})
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].table < specs[j].table })
	return specs, nil
}

func modelTable(st *ast.StructType) (string, bool) {
	for _, f := range st.Fields.List {
		if len(f.Names) != 0 {
			continue
		}
		if !isModelField(f) {
			continue
		}
		tag := reflect.StructTag("")
		if f.Tag != nil {
			if s, err := strconv.Unquote(f.Tag.Value); err == nil {
				tag = reflect.StructTag(s)
			}
		}
		db := tag.Get("db")
		if db == "" || db == "-" {
			continue
		}
		parts := strings.Fields(db)
		if len(parts) == 0 || strings.HasPrefix(parts[0], "ref:") {
			continue
		}
		return strings.ToLower(parts[0]), true
	}
	return "", false
}

func isModelField(f *ast.Field) bool {
	if len(f.Names) != 0 {
		return false
	}
	if sel, ok := f.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Model" {
		return true
	}
	if id, ok := f.Type.(*ast.Ident); ok && id.Name == "Model" {
		return true
	}
	return false
}

func collectColumns(st *ast.StructType, cols *[]colSpec) {
	for _, f := range st.Fields.List {
		// embedded data.Model marker
		if len(f.Names) == 0 && isModelField(f) {
			continue
		}

		tag := reflect.StructTag("")
		if f.Tag != nil {
			if s, err := strconv.Unquote(f.Tag.Value); err == nil {
				tag = reflect.StructTag(s)
			}
		}

		db := tag.Get("db")

		isEmbedded := len(f.Names) == 0
		if db == "" && isEmbedded {
			if st2, ok := structType(f.Type); ok {
				collectColumns(st2, cols)
				continue
			}
		}
		if db == "" {
			continue
		}
		if db == "-" {
			continue
		}
		parts := strings.Fields(db)
		if len(parts) == 0 {
			continue
		}
		if strings.HasPrefix(parts[0], "ref:") {
			continue // relation, not a column
		}

		flags := map[string]bool{}
		for _, p := range parts[1:] {
			flags[p] = true
		}

		cs := colSpec{
			name:   strings.ToLower(parts[0]),
			expr:   f.Type,
			auto:   flags["auto"],
			index:  flags["index"],
			unique: flags["unique"],
		}

		names := f.Names
		if len(names) == 0 {
			*cols = append(*cols, cs)
			continue
		}
		for range names {
			*cols = append(*cols, cs)
		}
	}
}

func structType(expr ast.Expr) (*ast.StructType, bool) {
	switch e := expr.(type) {
	case *ast.StructType:
		return e, true
	case *ast.StarExpr:
		if s, ok := e.X.(*ast.StructType); ok {
			return s, true
		}
	}
	return nil, false
}

func renderDDL(spec modelSpec, dialect string) string {
	var lines []string
	var indexes []string
	for _, c := range spec.cols {
		if c.auto {
			if dialect == "postgres" {
				lines = append(lines, c.name+" BIGSERIAL PRIMARY KEY")
			} else {
				lines = append(lines, c.name+" INTEGER PRIMARY KEY")
			}
			continue
		}
		typ, err := scalarSQL(c.expr, dialect)
		if err != nil {
			continue
		}
		line := c.name + " " + typ
		if !isPointer(c.expr) {
			line += " NOT NULL"
		}
		lines = append(lines, line)
		if c.unique {
			indexes = append(indexes, fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS uniq_%s_%s ON %s (%s)", spec.table, c.name, spec.table, c.name))
		} else if c.index {
			indexes = append(indexes, fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s)", spec.table, c.name, spec.table, c.name))
		}
	}
	create := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)", spec.table, strings.Join(lines, ",\n  "))
	stmts := []string{create}
	stmts = append(stmts, indexes...)
	return strings.Join(stmts, ";\n\n") + ";"
}

func renderMigration(specs []modelSpec, dialect string) string {
	var blocks []string
	for _, s := range specs {
		blocks = append(blocks, renderDDL(s, dialect))
	}
	return strings.Join(blocks, "\n\n")
}

func scalarSQL(expr ast.Expr, dialect string) (string, error) {
	e := expr
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
		_ = s
	}
	// []byte / []uint8
	if arr, ok := e.(*ast.ArrayType); ok && arr.Len == nil {
		if id, ok := arr.Elt.(*ast.Ident); ok && (id.Name == "byte" || id.Name == "uint8") {
			return "BLOB", nil
		}
		return "", fmt.Errorf("unsupported slice column")
	}
	name := typeExprName(e)
	switch name {
	case "string":
		return "TEXT", nil
	case "bool":
		if dialect == "postgres" {
			return "BOOLEAN", nil
		}
		return "INTEGER", nil
	case "int":
		return "INTEGER", nil
	case "int8", "int16", "int32":
		return "INTEGER", nil
	case "int64":
		if dialect == "postgres" {
			return "BIGINT", nil
		}
		return "INTEGER", nil
	case "uint", "uint8", "uint16", "uint32":
		return "INTEGER", nil
	case "uint64":
		if dialect == "postgres" {
			return "BIGINT", nil
		}
		return "INTEGER", nil
	case "float32":
		return "REAL", nil
	case "float64":
		if dialect == "postgres" {
			return "DOUBLE PRECISION", nil
		}
		return "REAL", nil
	case "Time":
		if dialect == "postgres" {
			return "TIMESTAMPTZ", nil
		}
		return "TEXT", nil
	case "byte":
		return "BLOB", nil
	}
	return "", fmt.Errorf("unsupported type %s", name)
}

func typeExprName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

func isPointer(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}
