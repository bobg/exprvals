package main

func f() bool {
	var x bool
	return x // want complete: `false`
}
