package main

func f() string {
	x, _ := g()
	return x
}

func g() (string, error) {
	return "hello", nil
}
