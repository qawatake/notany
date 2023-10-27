package notany

import (
	"fmt"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/packages"
)

func SetPanics(f func(v any)) {
	panics = f
}

func NewAnalyzer(dir string, targets ...Target) *analysis.Analyzer {
	apkgs := loadAdditionalPackages(dir, targets, nil)
	m := make(map[string]*packages.Package)
	for _, p := range apkgs {
		fmt.Println(p.PkgPath)
		m[p.PkgPath] = p
	}
	parsed := parseTargets(m, targets)
	r := &runner{
		parsed: parsed,
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
