package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"

	"github.com/qawatake/notany/internal/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

func main() {
	var t = Target{
		PkgPath:  "github.com/qawatake/notany/internal/example",
		FuncName: "FuncWithAnyTypeArg",
		ArgPos:   0,
		Allowed: []Allowed{
			{
				PkgPath:  "",
				TypeName: "int",
			},
			{
				PkgPath:  "fmt",
				TypeName: "Stringer",
			},
			{
				PkgPath:  "github.com/qawatake/notany/internal/example",
				TypeName: "AllowedType",
			},
			{
				PkgPath:  "github.com/qawatake/notany/internal/example/hoge",
				TypeName: "Hoger",
			},
		},
	}
	pkgs := loadPackages([]Target{t})
	targets := parseTargets(pkgs, []Target{t})
	for _, pkg := range pkgs {
		inspect := inspector.New(pkg.Syntax)
		pass := &analysis.Pass{
			TypesInfo: pkg.TypesInfo,
			Pkg:       pkg.Types,
			Fset:      pkg.Fset,
		}
		inspect.Preorder(nil, func(n ast.Node) {
			switch n := n.(type) {
			case *ast.CallExpr:
				if result := toBeReported(pass, targets, n); result != nil {
					// position := pass.Fset.Position(n.Pos())
					// fmt.Printf("👀%s:%d:%d %s is not allowed for the %dth arg of %s\n", pass.Fset.File(n.Pos()).Name(), position.Line, position.Column, result.ArgType, result.ArgPos+1, result.Func)
					// pass.Reportf(n.Pos(), "%s is not allowed for the %dth arg of %s", result.ArgType, result.ArgPos+1, result.Func)
					reportf(pass.Fset, n.Pos(), "%s is not allowed for the %dth arg of %s", result.ArgType, result.ArgPos+1, result.Func)
				}
			}
		})
	}
}

// Reportf is a helper function that reports a Diagnostic using the
// specified position and formatted error message.
// func (pass *Pass) Reportf(pos token.Pos, format string, args ...interface{}) {
func reportf(fset *token.FileSet, pos token.Pos, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	diag := analysis.Diagnostic{Pos: pos, Message: msg}
	posn := fset.Position(diag.Pos)
	fmt.Fprintf(os.Stderr, "%s: %s\n", posn, diag.Message)
}

// func PrintPlain(fset *token.FileSet, diag analysis.Diagnostic) {
// 	posn := fset.Position(diag.Pos)
// 	fmt.Fprintf(os.Stderr, "%s: %s\n", posn, diag.Message)

// 	// -c=N: show offending line plus N lines of context.
// 	if Context >= 0 {
// 		posn := fset.Position(diag.Pos)
// 		end := fset.Position(diag.End)
// 		if !end.IsValid() {
// 			end = posn
// 		}
// 		data, _ := os.ReadFile(posn.Filename)
// 		lines := strings.Split(string(data), "\n")
// 		for i := posn.Line - Context; i <= end.Line+Context; i++ {
// 			if 1 <= i && i <= len(lines) {
// 				fmt.Fprintf(os.Stderr, "%d\t%s\n", i, lines[i-1])
// 			}
// 		}
// 	}
// }

func loadPackages(targets []Target) []*packages.Package {
	length := 0
	for _, t := range targets {
		length += len(t.Allowed)
	}
	pkgPaths := make([]string, 0, length+1)
	pkgPaths = append(pkgPaths, "./...")
	for _, t := range targets {
		for _, a := range t.Allowed {
			pkgPaths = append(pkgPaths, a.PkgPath)
		}
	}
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
	}, pkgPaths...)
	if err != nil {
		panic(err)
	}
	return pkgs
}

const name = "notany"
const doc = "notany limits possible types for arguments of any type"

// Target represents a pair of a function and an argument with allowed types.
type Target struct {
	// Package path of the target function
	PkgPath string
	// Name of the target function
	FuncName string
	// Position of argument any
	// ArgPos is 0-indexed
	ArgPos int
	// List of allowed types for the argument
	Allowed []Allowed
}

// Allowed is a type that is allowed for the argument.
type Allowed struct {
	// The path of the package that defines the type.
	// If the type is builtin, let it be an empty string.
	PkgPath string
	// The name of the type.
	TypeName string
}

func parseTargets(pkgs []*packages.Package, targets []Target) []*analysisTarget {
	var ret []*analysisTarget
	passes := make([]*analysis.Pass, 0, len(pkgs))
	for _, pkg := range pkgs {
		passes = append(passes, &analysis.Pass{
			TypesInfo: pkg.TypesInfo,
			Pkg:       pkg.Types,
		})
	}
	for _, t := range targets {
		allowed := make([]types.Type, 0, len(t.Allowed))
		for _, a := range t.Allowed {
			if a.PkgPath == "" {
				typ := types.Universe.Lookup(a.TypeName).Type()
				if typ == nil {
					continue
				}
				allowed = append(allowed, typ)
				// builtin alias
				switch typ {
				case types.Typ[types.Uint8]:
					// byteType != types.Typ[types.Byte]
					allowed = append(allowed, byteType)
				case types.Typ[types.Int32]:
					// runeType != types.Typ[types.Rune]
					allowed = append(allowed, runeType)
				case byteType:
					allowed = append(allowed, types.Typ[types.Uint8])
				case runeType:
					allowed = append(allowed, types.Typ[types.Int32])
				}
				continue
			}
			for _, pass := range passes {
				if t := analysisutil.TypeOf(pass, a.PkgPath, a.TypeName); t != nil {
					allowed = append(allowed, t)
					continue
				}
			}
		}
		for _, pass := range passes {
			f := objectOf(pass, t)
			if f != nil {
				ret = append(ret, &analysisTarget{
					Func:    objectOf(pass, t),
					ArgPos:  t.ArgPos,
					Allowed: allowed,
				})
				continue
			}
		}
	}
	return ret
}

type analysisTarget struct {
	Func    types.Object
	ArgPos  int
	Allowed []types.Type
}

func (a *analysisTarget) Allow(t types.Type) bool {
	for _, at := range a.Allowed {
		if types.Identical(t, at) {
			return true
		}
		if i, ok := at.Underlying().(*types.Interface); ok {
			if types.Implements(t, i) {
				return true
			}
		}
	}
	return false
}

var byteType = types.Universe.Lookup("byte").Type()
var runeType = types.Universe.Lookup("rune").Type()

func objectOf(pass *analysis.Pass, t Target) types.Object {
	// function
	if !strings.Contains(t.FuncName, ".") {
		return analysisutil.ObjectOf(pass, t.PkgPath, t.FuncName)
	}
	tt := strings.Split(t.FuncName, ".")
	if len(tt) != 2 {
		panics(fmt.Sprintf("invalid FuncName %s", t.FuncName))
	}
	// method
	recv := tt[0]
	method := tt[1]
	recvType := analysisutil.TypeOf(pass, t.PkgPath, recv)
	return analysisutil.MethodOf(recvType, method)
}

var panics = func(v any) { panic(v) }

// toBeReported reports whether the call expression n should be reported.
// If nill is returned, it means that n should not be reported.
func toBeReported(pass *analysis.Pass, targets []*analysisTarget, n *ast.CallExpr) *notAllowed {
	switch f := n.Fun.(type) {
	case *ast.Ident:
		return x(pass, targets, n, f)
	case *ast.SelectorExpr:
		return x(pass, targets, n, f.Sel)
	}
	return nil
}

func x(pass *analysis.Pass, targets []*analysisTarget, n *ast.CallExpr, f *ast.Ident) *notAllowed {
	obj, ok := pass.TypesInfo.ObjectOf(f).(*types.Func)
	if !ok {
		return nil
	}
	sig, _ := obj.Type().(*types.Signature)
	for _, t := range targets {
		if t.Func != obj {
			continue
		}
		switch {
		case !sig.Variadic():
			arg := n.Args[t.ArgPos]
			argType := pass.TypesInfo.Types[arg].Type
			if !t.Allow(argType) {
				return &notAllowed{
					ArgExpr: arg,
					ArgType: argType,
					ArgPos:  t.ArgPos,
					Func:    obj,
				}
			}
			continue
		case sig.Variadic():
			for p := t.ArgPos; p < len(n.Args); p++ {
				arg := n.Args[p]
				argType := pass.TypesInfo.Types[arg].Type
				if !t.Allow(argType) {
					return &notAllowed{
						ArgExpr: arg,
						ArgType: argType,
						ArgPos:  p,
						Func:    obj,
					}
				}
			}
			continue
		}
	}
	return nil
}

type notAllowed struct {
	ArgExpr ast.Expr
	ArgType types.Type
	ArgPos  int
	Func    *types.Func
}
