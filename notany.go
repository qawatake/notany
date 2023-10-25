package notany

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
	"sync"

	"github.com/qawatake/notany/internal/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
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

var m sync.RWMutex
var once sync.Once

func (r *runner) run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	// targets := parseTargets(pass, r.targets)

	m.Lock()
	once.Do(func() {
		allow = newAllower(r.targets)
	})
	m.Unlock()
	// fmt.Println(allow.unparsed)
	inspect.Preorder(nil, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CallExpr:
			if result := allow.Check(pass, n); result != nil {
				pass.Reportf(n.Pos(), "%s is not allowed for the %dth arg of %s", result.ArgType, result.ArgPos+1, result.Func)
			}
		}
	})

	return nil, nil
}

type allower struct {
	sync.RWMutex
	pkgs     map[string]*packages.Package
	parsed   []*analysisTarget
	unparsed map[*Target]struct{}
}

var allow *allower

func newAllower(targets []Target) *allower {
	a := &allower{
		pkgs: make(map[string]*packages.Package),
	}
	unparsed := make(map[*Target]struct{})
	for _, t := range targets {
		t := t
		unparsed[&t] = struct{}{}
	}
	a.unparsed = unparsed
	return a
}

func (a *allower) Check(pass *analysis.Pass, n *ast.CallExpr) *notAllowed {
	switch f := n.Fun.(type) {
	case *ast.Ident:
		return a.check(pass, n, f)
	case *ast.SelectorExpr:
		return a.check(pass, n, f.Sel)
	}
	return nil
}

func (a *allower) check(pass *analysis.Pass, n *ast.CallExpr, f *ast.Ident) *notAllowed {
	obj, ok := pass.TypesInfo.ObjectOf(f).(*types.Func)
	if !ok {
		return nil
	}
	sig, _ := obj.Type().(*types.Signature)
	m.RLock()
	for _, t := range a.parsed {
		if x := t.Allowx(pass, obj, sig, n); x != nil {
			m.RUnlock()
			return x
		}
	}
	m.RUnlock()

	parsed := make(map[*Target]struct{})
	m.Lock()
	defer m.Unlock()
	defer func() {
		for t := range parsed {
			delete(a.unparsed, t)
		}
	}()
	for t := range a.unparsed {
		x := a.parseTarget(pass, t)
		if x != nil {
			parsed[t] = struct{}{}
			a.parsed = append(a.parsed, x)
			if x := x.Allowx(pass, obj, sig, n); x != nil {
				return x
			}
		}
	}
	return nil
}

func (a *allower) parseTarget(pass *analysis.Pass, t *Target) *analysisTarget {
	allowed := make([]types.Type, 0, len(t.Allowed))
	for _, al := range t.Allowed {
		if al.PkgPath == "" {
			typ := types.Universe.Lookup(al.TypeName).Type()
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
		if t := analysisutil.TypeOf(pass, al.PkgPath, al.TypeName); t != nil {
			allowed = append(allowed, t)
			continue
		}
		if pkg, ok := a.pkgs[al.PkgPath]; ok {
			pass := &analysis.Pass{
				TypesInfo: pkg.TypesInfo,
				Pkg:       pkg.Types,
			}
			if t := analysisutil.TypeOf(pass, al.PkgPath, al.TypeName); t != nil {
				allowed = append(allowed, t)
			}
			continue
		}
		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedTypes | packages.NeedTypesInfo,
		}, al.PkgPath)
		if err != nil {
			panic(err)
		}
		for _, pkg := range pkgs {
			if pkg.Types == nil || pkg.Types.Path() != al.PkgPath {
				continue
			}
			a.pkgs[pkg.Types.Path()] = pkg
			pass := &analysis.Pass{
				TypesInfo: pkg.TypesInfo,
				Pkg:       pkg.Types,
			}
			if t := analysisutil.TypeOf(pass, al.PkgPath, al.TypeName); t != nil {
				allowed = append(allowed, t)
				continue
			}
		}
	}
	f := objectOf(pass, t)
	if f != nil {
		return &analysisTarget{
			Func:    f,
			ArgPos:  t.ArgPos,
			Allowed: allowed,
		}
	}
	return nil
}

type analysisTarget struct {
	Func    types.Object
	ArgPos  int
	Allowed []types.Type
}

func (t *analysisTarget) Allowx(pass *analysis.Pass, obj *types.Func, sig *types.Signature, n *ast.CallExpr) *notAllowed {
	if t.Func != obj {
		return nil
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
		return nil
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
		return nil
	}
	return nil
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

func objectOf(pass *analysis.Pass, t *Target) types.Object {
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
