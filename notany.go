package notany

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/qawatake/notany/internal/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

const name = "notany"
const doc = "notany limits possible types for arguments of any type"

var loadedPackages []*packages.Package

func loadPackages(dir string, pattern ...string) []*packages.Package {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  dir,
	}, pattern...)
	if err != nil {
		panic(err)
	}
	return pkgs
}

func loadAdditionalPackages(dir string, targets []Target, have []*packages.Package) []*packages.Package {
	m := make(map[string]struct{})
	for _, t := range targets {
		for _, a := range t.Allowed {
			m[a.PkgPath] = struct{}{}
		}
	}
	for _, p := range have {
		delete(m, p.PkgPath)
	}
	pkgPaths := make([]string, 0, len(m))
	for k := range m {
		pkgPaths = append(pkgPaths, k)
	}
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  dir,
	}, pkgPaths...)
	if err != nil {
		panic(err)
	}
	return pkgs
}

type runner struct {
	parsed []*analysisTarget
}

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

func run(targets []Target, pattern ...string) error {
	pkgs := loadPackages(".", pattern...)
	apkgs := loadAdditionalPackages(".", targets, pkgs)
	m := make(map[string]*packages.Package)
	for _, p := range pkgs {
		m[p.PkgPath] = p
	}
	for _, p := range apkgs {
		m[p.PkgPath] = p
	}
	parsed := parseTargets(m, targets)
	r := &runner{
		parsed: parsed,
	}

	for _, pkg := range pkgs {
		inspectx := inspector.New(pkg.Syntax)
		pass := &analysis.Pass{
			TypesInfo: pkg.TypesInfo,
			Pkg:       pkg.Types,
			ResultOf:  map[*analysis.Analyzer]interface{}{inspect.Analyzer: inspectx},
			Report: func(d analysis.Diagnostic) {
				fmt.Println(pkg.Fset.Position(d.Pos), d.Message)
			},
		}
		_, err := r.run(pass)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	inspect.Preorder(nil, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CallExpr:
			if result := toBeReported(pass, r.parsed, n); result != nil {
				pass.Reportf(n.Pos(), "%s is not allowed for the %dth arg of %s", result.ArgType, result.ArgPos+1, result.Func)
			}
		}
	})

	return nil, nil
}

func parseTargets(pkgs map[string]*packages.Package, targets []Target) []*analysisTarget {
	var ret []*analysisTarget
	for _, t := range targets {
		allowed := make(map[types.Type]struct{})
		for _, a := range t.Allowed {
			if a.PkgPath == "" {
				parsed := parsetAllowedGlobally(a)
				for x := range parsed {
					allowed[x] = struct{}{}
				}
			}
			if pkg, ok := pkgs[a.PkgPath]; ok && pkg != nil {
				if t := pkg.Types.Scope().Lookup(a.TypeName); t != nil {
					allowed[t.Type()] = struct{}{}
				}
			}
		}
		ret = append(ret, &analysisTarget{
			F:       func(pass *analysis.Pass) types.Object { return objectOf(pass, t) },
			ArgPos:  t.ArgPos,
			Allowed: allowed,
		})
	}
	return ret
}

func parsetAllowedGlobally(a Allowed) map[types.Type]struct{} {
	allowed := make(map[types.Type]struct{})
	typ := types.Universe.Lookup(a.TypeName).Type()
	allowed[typ] = struct{}{}
	// builtin alias
	switch typ {
	case types.Typ[types.Uint8]:
		// byteType != types.Typ[types.Byte]
		allowed[byteType] = struct{}{}
	case types.Typ[types.Int32]:
		// runeType != types.Typ[types.Rune]
		allowed[runeType] = struct{}{}
	case byteType:
		allowed[types.Typ[types.Uint8]] = struct{}{}
	case runeType:
		allowed[types.Typ[types.Int32]] = struct{}{}
	}
	return allowed
}

// func toAnalysisTargets(pass *analysis.Pass, targets []Target) []*analysisTarget {
// 	var ret []*analysisTarget
// 	for _, t := range targets {
// 		allowed := make(map[types.Type]struct{})
// 		for _, a := range t.Allowed {
// 			if a.PkgPath == "" {
// 				typ := types.Universe.Lookup(a.TypeName).Type()
// 				allowed[typ] = struct{}{}
// 				// builtin alias
// 				switch typ {
// 				case types.Typ[types.Uint8]:
// 					// byteType != types.Typ[types.Byte]
// 					allowed[byteType] = struct{}{}
// 				case types.Typ[types.Int32]:
// 					// runeType != types.Typ[types.Rune]
// 					allowed[runeType] = struct{}{}
// 				case byteType:
// 					allowed[types.Typ[types.Uint8]] = struct{}{}
// 				case runeType:
// 					allowed[types.Typ[types.Int32]] = struct{}{}
// 				}
// 				continue
// 			}
// 			if t := analysisutil.TypeOf(pass, a.PkgPath, a.TypeName); t != nil {
// 				allowed[t] = struct{}{}
// 			}
// 		}
// 		ret = append(ret, &analysisTarget{
// 			Func:    objectOf(pass, t),
// 			ArgPos:  t.ArgPos,
// 			Allowed: allowed,
// 		})
// 	}
// 	return ret
// }

type analysisTarget struct {
	// Func    types.Object
	F       func(pass *analysis.Pass) types.Object
	ArgPos  int
	Allowed map[types.Type]struct{}
}

func (a *analysisTarget) Allow(t types.Type) bool {
	fmt.Println(t)
	for at := range a.Allowed {
		fmt.Println(at)
		if types.Identical(t, at) {
			return true
		}
		if i, ok := at.Underlying().(*types.Interface); ok {
			fmt.Println(i)
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
		if t.F(pass) != obj {
			continue
		}
		switch {
		case !sig.Variadic():
			if len(n.Args) <= t.ArgPos {
				return nil
			}
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
