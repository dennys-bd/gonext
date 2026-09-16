package generate

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// facadeInfo is what inspectFacade learned about a domain facade: the
// parameter names bound to the three types Register needs, the byte
// offsets to splice at, and the imports already present.
type facadeInfo struct {
	api, db, logger         string
	retOffset, importOffset int
	parenImports            bool
	firstStmt               bool
	has                     map[string]bool
}

func facadeImports(m resourceModel) []string {
	base := m.Module + "/backend/" + m.Domain + "/internal/"
	return []string{base + "application", base + "infrastructure/postgres", base + "presentation"}
}

// wireError is the pre-flight failure for a facade the generator cannot
// edit; it names the reason and the three statements to add by hand,
// using whichever parameter names info already bound.
func wireError(rel string, info facadeInfo, m resourceModel, reason string) error {
	stmts := wireStatements(m, info)
	return fmt.Errorf("cannot wire %s into %s: %s\nadd to Register by hand:\n\t%s",
		m.Name, rel, reason, strings.Join(stmts[:], "\n\t"))
}

func wireStatements(m resourceModel, info facadeInfo) [3]string {
	return [3]string{
		m.Camel + "Repo := postgres.New" + m.Pascal + "Repository(" + info.db + ")",
		m.Camel + "Svc := application.New" + m.Pascal + "Service(" + m.Camel + "Repo)",
		"presentation.Register" + m.Pascal + "(" + info.api + ", " + m.Camel + "Svc, " + info.logger + ")",
	}
}

// inspectFacade parses src (the facade at root-relative rel) and reports
// where wireFacade will splice. Its error names the reason and the
// three statements to add by hand.
func inspectFacade(rel string, src []byte, m resourceModel) (facadeInfo, error) {
	info := facadeInfo{api: "api", db: "db", logger: "logger", has: map[string]bool{}}
	fail := func(reason string) (facadeInfo, error) {
		return facadeInfo{}, wireError(rel, info, m, reason)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
	if err != nil {
		return fail(err.Error())
	}

	// Imports: a parenthesised group takes new specs before its ")"; a
	// bare `import "x"` (or no import at all) gets a new group after it.
	info.importOffset = fset.Position(file.Name.End()).Offset
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gen.Specs {
			if imp, ok := spec.(*ast.ImportSpec); ok {
				info.has[strings.Trim(imp.Path.Value, `"`)] = true
			}
		}
		info.parenImports = gen.Lparen.IsValid()
		info.importOffset = fset.Position(gen.End()).Offset
		if info.parenImports {
			info.importOffset = fset.Position(gen.Rparen).Offset
		}
	}

	var register *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "Register" {
			register = fn
			break
		}
	}
	if register == nil {
		return fail("no func Register")
	}

	// Bind names as they are found so a failure's hand-add hint uses
	// the facade's own names for the parameters it does have.
	slots := map[string]*string{"huma.API": &info.api, "*bun.DB": &info.db, "*slog.Logger": &info.logger}
	found := map[string]bool{}
	for _, param := range register.Type.Params.List {
		typ := paramTypeName(param.Type)
		slot, ok := slots[typ]
		if !ok || found[typ] || len(param.Names) == 0 {
			continue
		}
		*slot = param.Names[0].Name
		found[typ] = true
	}
	for _, typ := range []string{"huma.API", "*bun.DB", "*slog.Logger"} {
		if !found[typ] {
			return fail("Register has no " + typ + " parameter")
		}
	}

	body := register.Body
	if body == nil || len(body.List) == 0 {
		return fail("Register does not end with return nil")
	}
	ret, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return fail("Register does not end with return nil")
	}
	if id, ok := ret.Results[0].(*ast.Ident); !ok || id.Name != "nil" {
		return fail("Register does not end with return nil")
	}
	info.retOffset = fset.Position(ret.Pos()).Offset
	info.firstStmt = len(body.List) == 1
	return info, nil
}

// paramTypeName renders pkg.Name or *pkg.Name by selector text only; no
// type-checking, so an aliased import is not recognised.
func paramTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + paramTypeName(t.X)
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// wireFacade returns src with the three wiring statements inserted
// before Register's final return nil and any missing imports added,
// formatted with go/format.
func wireFacade(src []byte, info facadeInfo, m resourceModel) ([]byte, error) {
	stmts := wireStatements(m, info)
	block := strings.Join(stmts[:], "\n\t") + "\n\t"
	if !info.firstStmt {
		block = "\n\t" + block
	}

	var missing []string
	for _, imp := range facadeImports(m) {
		if !info.has[imp] {
			missing = append(missing, "\t"+`"`+imp+`"`+"\n")
		}
	}
	var imports string
	switch {
	case len(missing) == 0:
	case info.parenImports:
		imports = strings.Join(missing, "")
	default:
		imports = "\n\nimport (\n" + strings.Join(missing, "") + ")"
	}

	// Splice from the highest offset down so earlier offsets stay valid.
	out := splice(src, info.retOffset, block)
	out = splice(out, info.importOffset, imports)
	formatted, err := format.Source(out)
	if err != nil {
		return nil, fmt.Errorf("formatting facade: %w", err)
	}
	return formatted, nil
}

func splice(src []byte, at int, text string) []byte {
	out := make([]byte, 0, len(src)+len(text))
	out = append(out, src[:at]...)
	out = append(out, text...)
	return append(out, src[at:]...)
}
