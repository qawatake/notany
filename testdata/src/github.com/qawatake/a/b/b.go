package b

import (
	"time"

	"github.com/qawatake/a"
)

func f() {
	a.Target(nil, 2, "3")        // ok
	a.Target(1, time.Now(), 3.3) // ok because time.Time implements fmt.Stringer.
	a.Target(1, 1.1, "3")        // want "not allowed"
	a.Target(1, nil, "3")        // want "not allowed"
	a.Target(1, hoge{}, "3")     // ok
}

var _ hoger = hoge{}

type hoger interface {
	Hoge()
}

type hoge struct{}

func (h hoge) Hoge() {}
