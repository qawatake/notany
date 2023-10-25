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
	// analysistest.Run(t, testdata, notany.NewAnalyzer(
	// 	notany.Target{
	// 		PkgPath:  "a",
	// 		FuncName: "Target",
	// 		ArgPos:   1,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "int",
	// 			},
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "string",
	// 			},
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "uint8",
	// 			},
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "int32",
	// 			},
	// 		},
	// 	},
	// 	notany.Target{
	// 		PkgPath:  "a",
	// 		FuncName: "Target3",
	// 		ArgPos:   1,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "rune",
	// 			},
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "byte",
	// 			},
	// 		},
	// 	},
	// 	notany.Target{
	// 		PkgPath:  "a",
	// 		FuncName: "Target4",
	// 		ArgPos:   1,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "int",
	// 			},
	// 			{
	// 				PkgPath:  "fmt",
	// 				TypeName: "Stringer",
	// 			},
	// 		},
	// 	},
	// 	notany.Target{
	// 		PkgPath:  "fmt",
	// 		FuncName: "Println",
	// 		ArgPos:   0,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "a",
	// 				TypeName: "MyInt",
	// 			},
	// 		},
	// 	},
	// 	notany.Target{
	// 		PkgPath:  "github.com/qawatake/example",
	// 		FuncName: "Any",
	// 		ArgPos:   0,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "string",
	// 			},
	// 			{
	// 				PkgPath:  "github.com/qawatake/example",
	// 				TypeName: "MyInt",
	// 			},
	// 		},
	// 	},
	// 	notany.Target{
	// 		PkgPath:  "a",
	// 		FuncName: "Struct.Scan",
	// 		ArgPos:   0,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "int",
	// 			},
	// 			{
	// 				PkgPath:  "a",
	// 				TypeName: "MyInt",
	// 			},
	// 		},
	// 	},
	// 	notany.Target{
	// 		PkgPath:  "a",
	// 		FuncName: "*Struct.Scan2",
	// 		ArgPos:   0,
	// 		Allowed: []notany.Allowed{
	// 			{
	// 				PkgPath:  "",
	// 				TypeName: "bool",
	// 			},
	// 		},
	// 	},
	// ), "a")

	analysistest.Run(t, testdata, notany.NewAnalyzer(
		notany.Target{
			PkgPath:  "github.com/qawatake/a",
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
				{
					PkgPath:  "github.com/qawatake/a/c",
					TypeName: "Hoger",
				},
			},
		},
	), "github.com/qawatake/a/...")
}

func TestAnalyzer_invalid_cfg(t *testing.T) {
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	called := false
	panics := func(v any) {
		called = true
	}
	notany.SetPanics(panics)
	analysistest.Run(t, testdata, notany.NewAnalyzer(
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
	if !called {
		t.Error("panic expected but not called")
	}
}
