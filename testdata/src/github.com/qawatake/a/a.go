package a

import (
	"time"

	"github.com/qawatake/a/b"
	_ "github.com/qawatake/a/c"
)

func f() {
	b.Target(nil, 2, "3")        // ok
	b.Target(1, time.Now(), 3.3) // ok because time.Time implements fmt.Stringer.
	b.Target(1, 1.1, "3")        // want "not allowed"
	b.Target(1, nil, "3")        // want "not allowed"
}
