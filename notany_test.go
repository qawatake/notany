package notany_test

import (
	"testing"

	"github.com/gostaticanalysis/testutil"
	"github.com/qawatake/notany"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
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
	), "a")
}
