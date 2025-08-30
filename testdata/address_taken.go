package main

func f() string {
	x := "hello"
	g(&x)
	return x // want incomplete: `"hello"`
}

func g(*string) {}
