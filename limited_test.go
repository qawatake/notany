package limited_test

import (
	"testing"

	"github.com/gostaticanalysis/testutil"
	"github.com/qawatake/limited"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	analysistest.Run(t, testdata, limited.NewAnalyzer(
		limited.Target{
			PkgPath:  "a",
			FuncName: "Target1",
			ArgPos:   1,
		},
	), "a")
}
