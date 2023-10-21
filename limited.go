package limited

import (
	"go/ast"
	"go/types"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const name = "limited"
const doc = "limited limits possible types for argument of type any"

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

type Allowed struct {
	PkgPath  string
	TypeName string
}

func (r *runner) run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	targets := toAnalysisTargets(pass, r.targets)

	inspect.Preorder(nil, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CallExpr:
			if !allow(pass, targets, n) {
				pass.Reportf(n.Pos(), "not allowed")
			}
		}
	})

	return nil, nil
}

type analysisTarget struct {
	Func    types.Type
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
			Func:    analysisutil.TypeOf(pass, t.PkgPath, t.FuncName),
			ArgPos:  t.ArgPos,
			Allowed: allowed,
		})
	}
	return ret
}

func allow(pass *analysis.Pass, targets []*analysisTarget, n *ast.CallExpr) bool {
	for _, t := range targets {
		if !types.Identical(t.Func, pass.TypesInfo.Types[n.Fun].Type) {
			continue
		}
		if len(n.Args) <= t.ArgPos {
			continue
		}
		arg := n.Args[t.ArgPos]
		argType := pass.TypesInfo.Types[arg].Type
		if _, ok := t.Allowed[argType]; !ok {
			return false
		}
	}
	return true
}
