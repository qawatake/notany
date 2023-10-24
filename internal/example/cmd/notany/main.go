package main

import (
	"github.com/qawatake/notany"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(
		notany.NewAnalyzer(
			notany.Target{
				PkgPath:  "github.com/qawatake/notany/example",
				FuncName: "FuncWithAnyTypeArg",
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
						PkgPath:  "github.com/qawatake/notany/example",
						TypeName: "AllowedType",
					},
				},
			},
		),
	)
}
