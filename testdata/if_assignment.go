package main

func f() string {
	x := "hello"
	if true {
		x = "goodbye"
	}
	return x
}
