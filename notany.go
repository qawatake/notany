package notany

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/qawatake/notany/internal/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const name = "notany"
const doc = "notany limits possible types for arguments of any type"
const url = "https://pkg.go.dev/github.com/qawatake/notany"

func NewAnalyzer(targets ...Target) *analysis.Analyzer {
	r := &runner{
		targets: targets,
	}
	return &analysis.Analyzer{
		Name: name,
		Doc:  doc,
		URL:  url,
		Run:  r.run,
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}
}

type runner struct {
	targets []Target
}

// Target represents a pair of a function and a list of arguments with allowed types.
type Target struct {
	// Package path of the target function (or method).
	PkgPath string
	// Name of the target function (or method).
	FuncName string
	// Position of argument of type any.
	// ArgPos is 0-indexed.
	ArgPos int
	// List of allowed types for the argument.
	Allowed []Allowed
}

// Allowed represents a type that is allowed for the argument.
type Allowed struct {
	// The path of the package that defines the type.
	// If the type is builtin, let it be an empty string.
	PkgPath string
	// The name of the type.
	TypeName string
}

func (r *runner) run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	targets, err := toAnalysisTargets(pass, r.targets)
	if err != nil {
		return nil, err
	}

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

func toAnalysisTargets(pass *analysis.Pass, targets []Target) ([]*analysisTarget, error) {
	ret := make([]*analysisTarget, 0, len(targets))
	for _, t := range targets {
		t := t
		ft, err := funcObjectOf(pass, t)
		if err != nil {
			if !errors.Is(err, targetNotFound) {
				return nil, err
			}
			continue
		}
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
			if t := analysisutil.TypeOfBFS(pass.Pkg, a.PkgPath, a.TypeName); t != nil {
				allowed[t] = struct{}{}
				continue
			}
			return nil, newErrIdentNotFound(pass.Pkg.Path(), a.PkgPath, a.TypeName)
		}
		a := &analysisTarget{
			Func:    ft,
			ArgPos:  t.ArgPos,
			Allowed: allowed,
		}
		if err := a.validate(); err != nil {
			return nil, err
		}
		ret = append(ret, a)
	}
	return ret, nil
}

type analysisTarget struct {
	Func    *types.Func
	ArgPos  int
	Allowed map[types.Type]struct{}
}

func (a *analysisTarget) validate() error {
	if a.Func == nil || a.Func == (*types.Func)(nil) {
		return nil
	}
	sig, ok := a.Func.Type().(*types.Signature)
	if !ok {
		return nil
	}
	if sig.Params().Len() <= a.ArgPos {
		return newErrArgPosOutOfRange(a.Func.Pkg().Path(), a.Func.Name(), a.ArgPos)
	}
	return nil
}

func (a *analysisTarget) Allow(t types.Type) bool {
	if _, ok := a.Allowed[t]; ok {
		return true
	}
	for at := range a.Allowed {
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

func funcObjectOf(pass *analysis.Pass, t Target) (*types.Func, error) {
	// function
	if !strings.Contains(t.FuncName, ".") {
		obj := analysisutil.ObjectOf(pass, t.PkgPath, t.FuncName)
		if obj == nil {
			// not found is ok because func need not to be called.
			return nil, targetNotFound
		}
		ft, ok := obj.(*types.Func)
		if !ok {
			return nil, newErrNotFunc(t.PkgPath, t.FuncName)
		}
		return ft, nil
	}
	tt := strings.Split(t.FuncName, ".")
	if len(tt) != 2 {
		return nil, newErrInvalidFuncName(t.FuncName)
	}
	// method
	recv := tt[0]
	method := tt[1]
	recvType := analysisutil.TypeOf(pass, t.PkgPath, recv)
	if recvType == nil {
		// not found is ok because method need not to be called.
		return nil, targetNotFound
	}
	m := analysisutil.MethodOf(recvType, method)
	if m == nil {
		return nil, newErrNotMethod(t.PkgPath, recv, method)
	}
	return m, nil
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

var targetNotFound = errors.New("target not found")

type notAllowed struct {
	ArgExpr ast.Expr
	ArgType types.Type
	ArgPos  int
	Func    *types.Func
}

type errArgPosOutOfRange struct {
	PkgPath  string
	FuncName string
	ArgPos   int
}

func newErrArgPosOutOfRange(pkgPath, funcName string, argPos int) errArgPosOutOfRange {
	return errArgPosOutOfRange{
		PkgPath:  pkgPath,
		FuncName: funcName,
		ArgPos:   argPos,
	}
}

func (e errArgPosOutOfRange) Error() string {
	return fmt.Sprintf("ArgPos %d is out of range for %s.%s", e.ArgPos, e.PkgPath, e.FuncName)
}

type errInvalidFuncName struct {
	FuncName string
}

func newErrInvalidFuncName(funcName string) errInvalidFuncName {
	return errInvalidFuncName{
		FuncName: funcName,
	}
}

func (e errInvalidFuncName) Error() string {
	return fmt.Sprintf("invalid FuncName %s", e.FuncName)
}

type errIdentNotFound struct {
	FromPkgPath string
	PkgPath     string
	Name        string
}

func newErrIdentNotFound(fromPkgPath, pkgPath, name string) errIdentNotFound {
	return errIdentNotFound{
		FromPkgPath: fromPkgPath,
		PkgPath:     pkgPath,
		Name:        name,
	}
}

func (e errIdentNotFound) Error() string {
	return fmt.Sprintf("%[1]s.%[2]s is not found in %[3]s or its imports. Import %[1]s to %[3]s", e.PkgPath, e.Name, e.FromPkgPath)
}

type errNotFunc struct {
	PkgPath  string
	FuncName string
}

func newErrNotFunc(pkgPath, funcName string) errNotFunc {
	return errNotFunc{
		PkgPath:  pkgPath,
		FuncName: funcName,
	}
}

func (e errNotFunc) Error() string {
	return fmt.Sprintf("%s.%s is not a function.", e.PkgPath, e.FuncName)
}

type errNotMethod struct {
	PkgPath    string
	Recv       string
	MethodName string
}

func newErrNotMethod(pkgPath, recv, methodName string) errNotMethod {
	return errNotMethod{
		PkgPath:    pkgPath,
		Recv:       recv,
		MethodName: methodName,
	}
}

func (e errNotMethod) Error() string {
	return fmt.Sprintf("%s.%s.%s is not a method.", e.PkgPath, e.MethodName, e.Recv)
}
