package main

func f() int {
	var x int
	return x // want complete: `0`
}
