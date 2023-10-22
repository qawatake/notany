package notany

func SetPanics(f func(v any)) {
	panics = f
}
