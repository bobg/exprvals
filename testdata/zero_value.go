package main

func f() string {
	var x string
	return x // want complete: `""`
}
