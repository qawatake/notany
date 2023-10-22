package a

import (
	"fmt"

	"github.com/qawatake/example"
)

func f() {
	// limited
	Target(nil, 2, "3")    // ok
	Target(1, "2", 3.3)    // ok
	Target(1, 1.1, "3")    // want "not allowed"
	Target(1, nil, "3")    // want "not allowed"
	Target(1, str("2"), 3) // ok because str is an alias of string

	// builtin alias
	Target(1, uint8(1), 3)  // ok
	Target(1, byte(1), 3)   // ok because byte is an alias of uint8
	Target(1, int32(1), 3)  // ok
	Target(1, rune(1), 3)   // ok because rune is an alias of int32
	Target3(1, int32(1), 3) // ok
	Target(1, byte(1), 3)   // ok

	// not limited
	Target2(nil, 2, "3") // ok
	Target2(1, "2", 3.3) // ok
	Target2(1, 1.1, "3") // ok
	Target2(1, nil, "3") // ok

	// variadic
	fmt.Println(1)                  // want "not allowed"
	fmt.Println(MyInt(1))           // ok
	fmt.Println(MyInt(1), 1)        // want "not allowed"
	fmt.Println(MyInt(1), MyInt(1)) // ok

	// third party package
	example.Any("ok")             // ok because string is allowed.
	example.Any(1)                // want "not allowed"
	example.Any(example.MyInt(1)) // ok because example.MyInt is allowed.
	example.Any(MyInt(1))         // want "not allowed"

	// method
	var s Struct
	s.Scan(1)        // ok because int is allowed.
	s.Scan(MyInt(1)) // ok because MyInt is allowed.
	s.Scan("bad")    // want "not allowed"

	// method of pointer receiver
	s.Scan2(true) // ok because bool is allowed.
	s.Scan2(nil)  // want "not allowed"

	HigherOrder()()
}

// b must be either int, string, uint8, or int32.
func Target(a any, b any, c any) {}

type str = string

// b can be any type.
func Target2(a any, b any, c any) {}

// b can be rune or byte.
func Target3(a any, b any, c any) {}

type MyInt int

type Struct struct{}

// v must be either MyInt or int
func (s Struct) Scan(v any) {}

// v must be bool
func (s *Struct) Scan2(v any) {}

func HigherOrder() func() { return func() {} }
