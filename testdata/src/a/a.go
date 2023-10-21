package a

import "fmt"

func f() {
	Target1(1, 2, "3")   // ok
	Target1(1, "2", "3") // ok
	Target1(1, 1.1, "3") // want "not allowed"
	Target1(1, nil, "3") // want "not allowed"

	fmt.Println(1)        // want "not allowed"
	fmt.Println(MyInt(1)) // ok
}

// a must be either int or string
func Target1(i int, a any, s string) {}

type MyInt int
