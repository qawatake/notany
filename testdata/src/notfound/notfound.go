package notfound

import "time"

func f() {
	Target(1) // ok
	// fmt is not imported.
	Target(time.Now()) // want "not allowed"
}

// v must be int or fmt.Stringer.
func Target(v any) {}
