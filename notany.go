package notany

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const name = "notany"
const doc = "notany limits possible types for arguments of any type"

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
				continue
			}
			allowed[analysisutil.TypeOf(pass, a.PkgPath, a.TypeName)] = struct{}{}
		}
		ret = append(ret, &analysisTarget{
			Func:    objectOf(pass, t),
			ArgPos:  t.ArgPos,
			Allowed: allowed,
		})
	}
	return ret
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
