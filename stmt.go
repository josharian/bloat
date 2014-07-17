package main

import (
	"go/ast"
	"go/token"
)

// closeStmt generates func() { s }() given s, when safe to do so.
func closeStmt(s ast.Stmt) ast.Stmt {

	if s == nil {
		return nil
	}

	switch s := s.(type) {
	case *ast.CaseClause, *ast.CommClause, *ast.DeclStmt, *ast.LabeledStmt:
		return s
	case *ast.AssignStmt:
		if s.Tok == token.DEFINE {
			return s
		}
	case *ast.ForStmt:
		if s.Init == nil && s.Post == nil && s.Cond == nil {
			// Loop forever. Detected by compiler as legit alternative
			// to trailing return, so leave. We could try harder.
			return s
		}
	case *ast.ExprStmt:
		// Leave alone panics; see ForStmt comment above.
		if call, ok := s.X.(*ast.CallExpr); ok && len(call.Args) == 1 {
			if fun, ok := call.Fun.(*ast.Ident); ok && fun.Name == "panic" {
				return s
			}
		}
	}

	safe := true
	ast.Inspect(s, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ReturnStmt, *ast.BranchStmt, *ast.DeferStmt:
			safe = false
		case *ast.CallExpr:
			// Leave recovers alone; they must be called directly.
			if len(n.Args) != 0 {
				break
			}
			if fun, ok := n.Fun.(*ast.Ident); ok && fun.Name == "recover" {
				safe = false
			}
		}
		return safe
	})
	if !safe {
		return s
	}

	f := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{s},
				},
			},
		},
	}

	return f
}
