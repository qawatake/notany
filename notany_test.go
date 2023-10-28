package notany_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/gostaticanalysis/testutil"
	"github.com/qawatake/notany"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	analysistest.Run(t, testdata, notany.NewAnalyzer(
		notany.Target{
			PkgPath:  "a",
			FuncName: "Target",
			ArgPos:   1,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
				{
					PkgPath:  "",
					TypeName: "string",
				},
				{
					PkgPath:  "",
					TypeName: "uint8",
				},
				{
					PkgPath:  "",
					TypeName: "int32",
				},
			},
		},
		notany.Target{
			PkgPath:  "a",
			FuncName: "Target3",
			ArgPos:   1,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "rune",
				},
				{
					PkgPath:  "",
					TypeName: "byte",
				},
			},
		},
		notany.Target{
			PkgPath:  "a",
			FuncName: "Target4",
			ArgPos:   1,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
				{
					PkgPath:  "fmt",
					TypeName: "Stringer",
				},
			},
		},
		notany.Target{
			PkgPath:  "fmt",
			FuncName: "Println",
			ArgPos:   0,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "a",
					TypeName: "MyInt",
				},
			},
		},
		notany.Target{
			PkgPath:  "github.com/qawatake/example",
			FuncName: "Any",
			ArgPos:   0,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "string",
				},
				{
					PkgPath:  "github.com/qawatake/example",
					TypeName: "MyInt",
				},
			},
		},
		notany.Target{
			PkgPath:  "a",
			FuncName: "Struct.Scan",
			ArgPos:   0,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
				{
					PkgPath:  "a",
					TypeName: "MyInt",
				},
			},
		},
		notany.Target{
			PkgPath:  "a",
			FuncName: "*Struct.Scan2",
			ArgPos:   0,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "bool",
				},
			},
		},
	), "a")

}

func TestAnalyzer_pkgpath_different_from_pkgname(t *testing.T) {
	t.Parallel()
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	analysistest.Run(t, testdata, notany.NewAnalyzer(
		notany.Target{
			PkgPath:  "github.com/qawatake/a/b",
			FuncName: "Target",
			ArgPos:   1,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
				{
					PkgPath:  "fmt",
					TypeName: "Stringer",
				},
			},
		},
	), "github.com/qawatake/a")
}

func TestAnalyzer_out_of_range(t *testing.T) {
	t.Parallel()
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	treporter := NewAnalysisErrorReporter(1)
	analysistest.Run(treporter, testdata, notany.NewAnalyzer(
		notany.Target{
			PkgPath:  "oor",
			FuncName: "OutOfRange",
			// ↓ out of range
			ArgPos: 1,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
			},
		}), "oor")
	errs := treporter.Errors()
	want := notany.ErrArgPosOutOfRange{
		PkgPath:  "oor",
		FuncName: "OutOfRange",
		ArgPos:   1,
	}
	if len(errs) != 1 {
		t.Errorf("err expected but not found: %v", want)
	}
	if !errors.Is(errs[0], want) {
		t.Errorf("got %v, want %v", errs[0], want)
	}
}

func TestAnalyzer_invalid_func_name(t *testing.T) {
	t.Parallel()
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	treporter := NewAnalysisErrorReporter(1)
	analysistest.Run(treporter, testdata, notany.NewAnalyzer(
		notany.Target{
			PkgPath:  "a",
			FuncName: ".Struct.Scan", // too much periods
			ArgPos:   0,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
				{
					PkgPath:  "a",
					TypeName: "MyInt",
				},
			},
		}), "empty")
	errs := treporter.Errors()
	want := notany.ErrInvalidFuncName{
		FuncName: ".Struct.Scan",
	}
	if len(errs) != 1 {
		t.Errorf("err expected but not found: %v", want)
	}
	if !errors.Is(errs[0], want) {
		t.Errorf("got %v, want %v", errs[0], want)
	}
}

func TestAnalyzer_not_found_allowed(t *testing.T) {
	t.Parallel()
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	treporter := NewAnalysisErrorReporter(1)
	analysistest.Run(treporter, testdata, notany.NewAnalyzer(
		notany.Target{
			PkgPath:  "notfound",
			FuncName: "Target",
			ArgPos:   0,
			Allowed: []notany.Allowed{
				{
					PkgPath:  "",
					TypeName: "int",
				},
				// not found
				{
					PkgPath:  "fmt",
					TypeName: "Stringer",
				},
			},
		}), "notfound")
	errs := treporter.Errors()
	want := notany.ErrIdentNotFound{
		FromPkgPath: "notfound",
		PkgPath:     "fmt",
		Name:        "Stringer",
	}
	if len(errs) != 1 {
		t.Errorf("err expected but not found: %v", want)
	}
	if !errors.Is(errs[0], want) {
		t.Errorf("got %v, want %v", errs[0], want)
	}
}

var _ analysistest.Testing = (*analysisErrorReporter)(nil)

type analysisErrorReporter struct {
	sync.RWMutex
	errs []error
}

func NewAnalysisErrorReporter(expected int) *analysisErrorReporter {
	return &analysisErrorReporter{
		errs: make([]error, 0, expected),
	}
}

func (r *analysisErrorReporter) Errorf(format string, args ...any) {
	errs := make([]error, 0, len(args))
	for _, arg := range args {
		if err, ok := arg.(error); ok {
			errs = append(errs, err)
		}
	}
	errs = append(errs, fmt.Errorf(format, args...))
	r.Lock()
	defer r.Unlock()
	r.errs = append(r.errs, errors.Join(errs...))
}

func (r *analysisErrorReporter) Errors() []error {
	r.RLock()
	defer r.RUnlock()
	return r.errs[:]
}
