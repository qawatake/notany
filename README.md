# notany

[![Go Reference](https://pkg.go.dev/badge/github.com/qawatake/notany.svg)](https://pkg.go.dev/github.com/qawatake/notany)
[![test](https://github.com/qawatake/notany/actions/workflows/test.yaml/badge.svg)](https://github.com/qawatake/notany/actions/workflows/test.yaml)
[![codecov](https://codecov.io/gh/qawatake/notany/graph/badge.svg?token=mjocIOzSRm)](https://codecov.io/gh/qawatake/notany)

Linter `notany` limits possible types for arguments of any type.

```go
// arg must be string, fmt.Stringer, or MyInt.
func FuncWithAnyTypeArg(arg any) {
  // ...
}

type AllowedType struct{}
```

```go
func main() {
  pkg.FuncWithAnyTypeArg("ok")          // ok
  pkg.FuncWithAnyTypeArg(time.Now())    // ok because time.Time implements fmt.Stringer
  pkg.FuncWithAnyTypeArg(AllowedType{}) // ok
  pkg.FuncWithAnyTypeArg(1.0)           // <- float64 is not allowed
  pkg.FuncWithAnyTypeArg(true)          // <- bool is not allowed
}
```

## How to use

Build your `notany` binary by writing `main.go` like below.

```go
package main

import (
  "github.com/qawatake/notany"
  "golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
  unitchecker.Main(
    notany.NewAnalyzer(
      notany.Target{
        PkgPath:  "pkg/in/which/target/func/is/defined",
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
            PkgPath:  "pkg/in/which/allowed/type/is/defined",
            TypeName: "AllowedType",
          },
        },
      },
    ),
  )
}
```

Then, run `go vet` with your `notany` binary.

```sh
go vet -vettool=/path/to/your/notany ./...
```
