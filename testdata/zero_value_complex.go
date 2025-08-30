package main

func f() complex128 {
	var x complex128
	return x // want complete: `0`
}
