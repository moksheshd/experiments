package main

import (
	"fmt"
)

func main() {
	fmt.Println("Data Types Example")
	var age int = 30
	fmt.Printf("Age is: %d\n", age)

	type IZ int
	var count IZ = 1
	fmt.Printf("Count: %d\n", count)

	var precision float32 = 10.1
	fmt.Println(precision)
	fmt.Println(int(precision))
}

func SomeFunction(a int, b string) (o1 bool, o2 int) {
	return true, -1
	// return
}
