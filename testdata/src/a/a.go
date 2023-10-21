package a

import "fmt"

func f() {
	// limited
	Target(nil, 2, "3") // ok
	Target(1, "2", 3.3) // ok
	Target(1, 1.1, "3") // want "not allowed"
	Target(1, nil, "3") // want "not allowed"

	// not limited
	Target2(nil, 2, "3") // ok
	Target2(1, "2", 3.3) // ok
	Target2(1, 1.1, "3") // ok
	Target2(1, nil, "3") // ok

	// variadic
	fmt.Println(1)           // want "not allowed"
	fmt.Println(MyInt(1))    // ok
	fmt.Println(MyInt(1), 1) // want "not allowed"
}

// b must be either int or string
func Target(a any, b any, c any) {}

// b can be any type
func Target2(a any, b any, c any) {}

type MyInt int
