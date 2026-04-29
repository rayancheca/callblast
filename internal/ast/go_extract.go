package ast

import (
	"crypto/sha1"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ExtractGoFile parses a Go source file and extracts all function definitions
// and call sites found within each function body.
func ExtractGoFile(filePath string, src []byte) ([]FunctionDef, []CallSite, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	var funcs []FunctionDef
	var calls []CallSite

	goast.Inspect(f, func(n goast.Node) bool {
		fd, ok := n.(*goast.FuncDecl)
		if !ok {
			return true
		}

		pos := fset.Position(fd.Pos())
		def := FunctionDef{
			Name:     fd.Name.Name,
			File:     filePath,
			Line:     pos.Line,
			Language: "go",
		}

		// Build signature string
		def.Signature = buildGoSignature(fd)
		def.Receiver = extractReceiver(fd)

		// Hash the body for change detection
		if fd.Body != nil {
			start := fset.Position(fd.Body.Lbrace)
			end := fset.Position(fd.Body.Rbrace)
			if start.Offset < end.Offset && end.Offset <= len(src) {
				body := src[start.Offset : end.Offset+1]
				def.BodyHash = sha1Hash(body)
			}
		}

		funcs = append(funcs, def)

		// Walk the body to find call sites
		callerQName := def.QualifiedName()
		if fd.Body != nil {
			goast.Inspect(fd.Body, func(inner goast.Node) bool {
				ce, ok := inner.(*goast.CallExpr)
				if !ok {
					return true
				}
				callPos := fset.Position(ce.Pos())
				calleeName := extractCalleeName(ce)
				if calleeName != "" {
					calls = append(calls, CallSite{
						CallerFunc: callerQName,
						CalleeFunc: calleeName,
						File:       filePath,
						Line:       callPos.Line,
					})
				}
				return true
			})
		}

		return true
	})

	return funcs, calls, nil
}

func buildGoSignature(fd *goast.FuncDecl) string {
	var sb strings.Builder
	sb.WriteString("func ")
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		sb.WriteString("(")
		for i, field := range fd.Recv.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			if len(field.Names) > 0 {
				sb.WriteString(field.Names[0].Name)
				sb.WriteString(" ")
			}
			sb.WriteString(exprToString(field.Type))
		}
		sb.WriteString(") ")
	}
	sb.WriteString(fd.Name.Name)
	sb.WriteString("(")
	if fd.Type.Params != nil {
		for i, field := range fd.Type.Params.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			for j, name := range field.Names {
				if j > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(name.Name)
				sb.WriteString(" ")
			}
			sb.WriteString(exprToString(field.Type))
		}
	}
	sb.WriteString(")")
	if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
		sb.WriteString(" ")
		if len(fd.Type.Results.List) > 1 {
			sb.WriteString("(")
		}
		for i, field := range fd.Type.Results.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprToString(field.Type))
		}
		if len(fd.Type.Results.List) > 1 {
			sb.WriteString(")")
		}
	}
	return sb.String()
}

func extractReceiver(fd *goast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	return exprToString(fd.Recv.List[0].Type)
}

func exprToString(expr goast.Expr) string {
	switch e := expr.(type) {
	case *goast.Ident:
		return e.Name
	case *goast.StarExpr:
		return "*" + exprToString(e.X)
	case *goast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *goast.ArrayType:
		return "[]" + exprToString(e.Elt)
	case *goast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *goast.InterfaceType:
		return "interface{}"
	case *goast.Ellipsis:
		return "..." + exprToString(e.Elt)
	case *goast.ChanType:
		return "chan " + exprToString(e.Value)
	default:
		return "?"
	}
}

func extractCalleeName(ce *goast.CallExpr) string {
	switch fn := ce.Fun.(type) {
	case *goast.Ident:
		return fn.Name
	case *goast.SelectorExpr:
		return fn.Sel.Name
	default:
		return ""
	}
}

func sha1Hash(data []byte) string {
	h := sha1.Sum(data)
	return fmt.Sprintf("%x", h)
}
