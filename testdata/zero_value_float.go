package main

func f() float64 {
	var x float64
	return x // want complete: `0`
}
