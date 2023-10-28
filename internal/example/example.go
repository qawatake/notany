package example

import (
	_ "fmt"
	"time"
)

// arg must be string, fmt.Stringer, or AllowedType.
func FuncWithAnyTypeArg(arg any) {
	// ...
}

type AllowedType struct{}

func Example() {
	FuncWithAnyTypeArg(10)            // ok
	FuncWithAnyTypeArg(time.Now())    // ok because time.Time implements fmt.Stringer
	FuncWithAnyTypeArg(AllowedType{}) // ok
	FuncWithAnyTypeArg(1.0)           // <- float64 is not allowed
	FuncWithAnyTypeArg(true)          // <- bool is not allowed
}
