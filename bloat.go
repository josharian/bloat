package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
)

type stmtsearch struct {
	par []ast.Node // statement parent nodes
}

func (s *stmtsearch) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause,
		*ast.ForStmt, *ast.IfStmt,
		*ast.LabeledStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		s.par = append(s.par, n)
	}
	return s
}

func replaceStmt(stmt ast.Stmt) ast.Stmt {
	return closeStmt(stmt)
}

func replaceAllStmts(ss []ast.Stmt) []ast.Stmt {
	t := make([]ast.Stmt, len(ss))
	for i := range ss {
		stmt := ss[i]
		t[i] = closeStmt(stmt)
	}
	return t
}

func (s *stmtsearch) update() {
	for _, n := range s.par {
		switch n := n.(type) {
		case *ast.BlockStmt:
			n.List = replaceAllStmts(n.List)
		case *ast.CaseClause:
			n.Body = replaceAllStmts(n.Body)
		case *ast.CommClause:
			n.Body = replaceAllStmts(n.Body)
		case *ast.ForStmt:
			n.Init = replaceStmt(n.Init)
			n.Post = replaceStmt(n.Post)
		case *ast.IfStmt:
			n.Init = replaceStmt(n.Init)
			n.Else = replaceStmt(n.Else)
		case *ast.LabeledStmt:
			n.Stmt = replaceStmt(n.Stmt)
		case *ast.SwitchStmt:
			n.Init = replaceStmt(n.Init)
		case *ast.TypeSwitchStmt:
			n.Init = replaceStmt(n.Init)
		}
	}
}

func main() {
	wd, err := os.Getwd()
	if len(os.Args) < 2 {
		fmt.Println("usage: bloat [packages]")
		os.Exit(2)
	}
	if err != nil {
		fatal(err)
	}
	var files []string
	for _, path := range os.Args[1:] {
		if path == "syscall" {
			// syscall is a snowflake. Leave it alone.
			continue
		}
		pkg, err := build.Import(path, wd, 0)
		if err != nil {
			fatal(err)
		}
		for _, file := range pkg.GoFiles {
			files = append(files, filepath.Join(pkg.Dir, file))
		}
	}

	for _, file := range files {
		// fmt.Println("Processing", file)
		fset := token.NewFileSet()
		// TODO: preserve comments, to avoid stripping build tags
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			fatal(err)
		}

		s := &stmtsearch{}
		ast.Walk(s, f)
		s.update()

		c, err := os.Create(file)
		if err != nil {
			fatal(err)
		}

		if err := printer.Fprint(c, fset, f); err != nil {
			fatal(err)
		}

		c.Close()
	}
}

func fatal(msg interface{}) {
	fmt.Println(msg)
	os.Exit(1)
}
