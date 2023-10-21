package a

func f() {
	Target1(1, 2, "3")   // ok
	Target1(1, "2", "3") // ok
	Target1(1, 1.1, "3") // want "not allowed"
	Target1(1, nil, "3") // want "not allowed"
}

// a must be either int or string
func Target1(i int, a any, s string) {}
