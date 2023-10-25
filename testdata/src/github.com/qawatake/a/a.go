package a

import "time"

func f() {
	Target(nil, 2, "3")        // ok
	Target(1, time.Now(), 3.3) // ok because time.Time implements fmt.Stringer.
	Target(1, 1.1, "3")        // want "not allowed"
	Target(1, nil, "3")        // want "not allowed"
}

// b must be either int or fmt.Stringer.
func Target(a any, b any, c any) {}

