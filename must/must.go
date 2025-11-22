package must

func Do(err error) {
	Be(err == nil, err)
}

func Do1[T any](x T, err error) T {
	Be(err == nil, err)
	return x
}

func Do2[T, U any](x T, y U, err error) (T, U) {
	Be(err == nil, err)
	return x, y
}

// Assert function, named must.Be for short
func Be(cond bool, msg any) {
	if !cond {
		panic(msg)
	}
}
