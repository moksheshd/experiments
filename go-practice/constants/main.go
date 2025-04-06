package main

import (
	"fmt"
)

const PI = 3.14159
const BEEF, TWO, C = "meat", 2, "veg"
const MONDAY, TUESDAY, WEDNESDAY, THURSDAY, FRIDAY, SATURDAY, SUNDAY = 1, 2, 3, 4, 5, 6, 7
const jan, FEB int = 1, 2

const (
	Unknown = 0
	Female  = 1
	Male    = 2
)

type Gender int

const (
	UNKNOWN Gender = iota
	FEMALE
	MALE
)

func main() {
	fmt.Println("This is the constants package")
	const PREFIX = "i_"

	fmt.Println(PREFIX)
	fmt.Println(Female)

	SomeFunction()

}

func SomeFunction() {
	fmt.Println(PI)
	fmt.Println("As PREFIX constant is defined in main. It cannot be used here")
}
