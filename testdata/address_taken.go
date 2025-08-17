package main

func f() string {
	x := "hello"
	g(&x)
	return x
}

func g(*string) {}
