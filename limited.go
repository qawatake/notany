package notany

import (
	"go/ast"
	"go/types"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const name = "notany"
const doc = "notany limits possible types for argument of type any"

func NewAnalyzer(targets ...Target) *analysis.Analyzer {
	r := &runner{
		targets: targets,
	}
	return &analysis.Analyzer{
		Name: name,
		Doc:  doc,
		Run:  r.run,
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}
}

type runner struct {
	targets []Target
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

func (r *runner) run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	targets := toAnalysisTargets(pass, r.targets)

	inspect.Preorder(nil, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CallExpr:
			if result := toBeReported(pass, targets, n); result != nil {
				pass.Reportf(n.Pos(), "%s is not allowed for the %dth arg of %s", result.ArgType, result.ArgPos+1, result.Func)
			}
		}
	})

	return nil, nil
}

type analysisTarget struct {
	Func    types.Object
	ArgPos  int
	Allowed map[types.Type]struct{}
}

func toAnalysisTargets(pass *analysis.Pass, targets []Target) []*analysisTarget {
	var ret []*analysisTarget
	for _, t := range targets {
		allowed := make(map[types.Type]struct{})
		for _, a := range t.Allowed {
			if a.PkgPath == "" {
				allowed[types.Universe.Lookup(a.TypeName).Type()] = struct{}{}
				continue
			}
			allowed[analysisutil.TypeOf(pass, a.PkgPath, a.TypeName)] = struct{}{}
		}
		ret = append(ret, &analysisTarget{
			Func:    analysisutil.ObjectOf(pass, t.PkgPath, t.FuncName),
			ArgPos:  t.ArgPos,
			Allowed: allowed,
		})
	}
	return ret
}

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
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return nil
	}
	for _, t := range targets {
		if t.Func != obj {
			continue
		}
		if len(n.Args) <= t.ArgPos {
			continue
		}
		if !sig.Variadic() {
			arg := n.Args[t.ArgPos]
			argType := pass.TypesInfo.Types[arg].Type
			if _, ok := t.Allowed[argType]; !ok {
				return &notAllowed{
					ArgExpr: arg,
					ArgType: argType,
					ArgPos:  t.ArgPos,
					Func:    obj,
				}
			}
			continue
		}
		for p := t.ArgPos; p < len(n.Args); p++ {
			arg := n.Args[p]
			argType := pass.TypesInfo.Types[arg].Type
			if _, ok := t.Allowed[argType]; !ok {
				return &notAllowed{
					ArgExpr: arg,
					ArgType: argType,
					ArgPos:  p,
					Func:    obj,
				}
			}
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
