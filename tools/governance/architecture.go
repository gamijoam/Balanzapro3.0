// Initial structural checks. No code from the inspected tree is executed.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const module = "github.com/gamijoam/Balanzapro3.0/backend/"

func isDB(s string) bool {
	return s == "database/sql" || strings.HasPrefix(s, "github.com/jackc/pgx/")
}
func allowedDB(p string) bool {
	return strings.Contains(p, "/adapters/") || strings.HasPrefix(p, "internal/platform/postgres/") ||
		strings.HasPrefix(p, "internal/platform/pgtest/") || strings.HasPrefix(p, "cmd/") ||
		p == "internal/organization/organization.go" // existing minimal scope store; do not expand
}
func core(p string) bool {
	return strings.Contains(p, "/domain/") || strings.Contains(p, "/application/") ||
		strings.HasPrefix(p, "internal/platform/qty/")
}
func layer(p string) string {
	for _, l := range []string{"domain", "application", "adapters", "jobs"} {
		if strings.Contains("/"+p, "/"+l+"/") {
			return l
		}
	}
	return ""
}
func forbidden(p, imp string) bool {
	if isDB(imp) && !allowedDB(p) {
		return true
	}
	if !strings.HasPrefix(imp, module) {
		return false
	}
	dst := strings.TrimPrefix(imp, module)
	srcLayer, dstLayer := layer(p), layer(dst+"/")
	if srcLayer == "domain" && (dstLayer == "application" || dstLayer == "adapters" || dstLayer == "jobs") {
		return true
	}
	if srcLayer == "application" && (dstLayer == "adapters" || dstLayer == "jobs") {
		return true
	}
	if core(p) && (strings.HasPrefix(dst, "internal/platform/postgres") || strings.HasPrefix(dst, "internal/platform/pgtest") || strings.HasPrefix(dst, "cmd/")) {
		return true
	}
	parts := strings.Split(p, "/")
	dparts := strings.Split(dst, "/")
	// Modules may not import another module's concrete adapters.
	return len(parts) > 2 && len(dparts) > 2 && parts[0] == "internal" && dparts[0] == "internal" &&
		parts[1] != dparts[1] && dstLayer == "adapters"
}
func inspect(root string) ([]string, error) {
	var violations []string
	graph := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == "testdata" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Do not follow symlinks from untrusted source trees.
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected Go symlink")
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("Go source parse failed: %s", rel)
		}
		pkg := module + filepath.ToSlash(filepath.Dir(rel))
		report := func(rule string) { violations = append(violations, rel+": "+rule) }
		for _, spec := range f.Imports {
			imp, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if forbidden(rel, imp) {
				report("forbidden layer/database import: " + imp)
			}
			if strings.HasPrefix(imp, module) {
				graph[pkg] = append(graph[pkg], imp)
			}
		}
		if core(rel) {
			ast.Inspect(f, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.Ident:
					if v.Name == "float32" || v.Name == "float64" {
						report("floating-point type in exact domain")
					}
				case *ast.BasicLit:
					if v.Kind == token.FLOAT {
						report("floating-point literal in exact domain")
					}
				case *ast.SelectorExpr:
					if v.Sel.Name == "NewFromFloat" || v.Sel.Name == "NewFromFloat32" || v.Sel.Name == "ParseFloat" || v.Sel.Name == "Float64" || v.Sel.Name == "InexactFloat64" {
						report("floating-point conversion in exact domain")
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	state := map[string]int{}
	var visit func(string)
	visit = func(p string) {
		if state[p] == 1 {
			violations = append(violations, "package dependency cycle: "+p)
			return
		}
		if state[p] == 2 {
			return
		}
		state[p] = 1
		for _, q := range graph[p] {
			visit(q)
		}
		state[p] = 2
	}
	for p := range graph {
		visit(p)
	}
	sort.Strings(violations)
	return violations, nil
}
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: architecture <backend-dir>")
		os.Exit(2)
	}
	v, err := inspect(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "architecture inspection failed")
		os.Exit(2)
	}
	for _, issue := range v {
		fmt.Printf("%q\n", issue)
	}
	if len(v) != 0 {
		os.Exit(1)
	}
	fmt.Println("Initial architecture rules: PASS")
}
