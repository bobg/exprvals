package main

func f() string {
	x, _ := g()
	return x // want complete: `"hello"`
}

func g() (string, error) {
	return "hello", nil
}
