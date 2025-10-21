// Package constants demonstrates constant declarations and usage in Go
package constants

import "fmt"

// Package-level constants
const (
	PI   = 3.14159
	BEEF = "meat"
	TWO  = 2
	C    = "veg"
)

// Days of the week as constants
const (
	MONDAY = 1 + iota
	TUESDAY
	WEDNESDAY
	THURSDAY
	FRIDAY
	SATURDAY
	SUNDAY
)

// Typed constants
const (
	Jan int = 1
	Feb int = 2
)

// Simple enumeration (old style)
const (
	Unknown = 0
	Female  = 1
	Male    = 2
)

// Gender custom type for better type safety
type Gender int

// Gender enumeration using iota
const (
	UNKNOWN Gender = iota
	FEMALE
	MALE
)

// DemoBasicConstants shows basic constant usage
func DemoBasicConstants() {
	fmt.Println("=== Basic Constants ===")
	fmt.Printf("PI: %f\n", PI)
	fmt.Printf("BEEF: %s\n", BEEF)
	fmt.Printf("TWO: %d\n", TWO)
	fmt.Printf("C: %s\n", C)
}

// DemoWeekdayConstants shows weekday constants
func DemoWeekdayConstants() {
	fmt.Println("\n=== Weekday Constants ===")
	fmt.Printf("MONDAY: %d\n", MONDAY)
	fmt.Printf("FRIDAY: %d\n", FRIDAY)
	fmt.Printf("SUNDAY: %d\n", SUNDAY)
}

// DemoTypedConstants shows typed vs untyped constants
func DemoTypedConstants() {
	fmt.Println("\n=== Typed Constants ===")
	fmt.Printf("Jan (typed int): %d\n", Jan)
	fmt.Printf("Feb (typed int): %d\n", Feb)
}

// DemoGenderEnum shows enumeration with custom types
func DemoGenderEnum() {
	fmt.Println("\n=== Gender Enumeration (iota) ===")
	fmt.Printf("UNKNOWN: %d\n", UNKNOWN)
	fmt.Printf("FEMALE: %d\n", FEMALE)
	fmt.Printf("MALE: %d\n", MALE)

	// Using the Gender type
	var g Gender = FEMALE
	fmt.Printf("Gender value: %d\n", g)
}

// DemoFunctionConstants shows function-scoped constants
func DemoFunctionConstants() {
	fmt.Println("\n=== Function-Scoped Constants ===")

	const PREFIX = "i_"
	fmt.Printf("PREFIX: %s\n", PREFIX)
	fmt.Printf("Female (old style): %d\n", Female)

	PrintPIValue()
}

// PrintPIValue demonstrates accessing package-level constants
func PrintPIValue() {
	fmt.Printf("PI from another function: %f\n", PI)
	fmt.Println("Note: PREFIX constant from DemoFunctionConstants cannot be used here")
}

// GetGenderString returns string representation of Gender
func GetGenderString(g Gender) string {
	switch g {
	case UNKNOWN:
		return "Unknown"
	case FEMALE:
		return "Female"
	case MALE:
		return "Male"
	default:
		return "Invalid"
	}
}

// DemoIotaAdvanced shows advanced iota usage
func DemoIotaAdvanced() {
	fmt.Println("\n=== Advanced iota Usage ===")

	const (
		_  = iota // Skip zero value
		KB = 1 << (10 * iota)
		MB
		GB
		TB
	)

	fmt.Printf("KB: %d bytes\n", KB)
	fmt.Printf("MB: %d bytes\n", MB)
	fmt.Printf("GB: %d bytes\n", GB)
	fmt.Printf("TB: %d bytes\n", TB)
}

// FileMode represents file permission modes
type FileMode uint32

// File mode constants
const (
	ModeRead FileMode = 1 << iota
	ModeWrite
	ModeExecute
)

// DemoFileModes shows bitwise constants with iota
func DemoFileModes() {
	fmt.Println("\n=== File Mode Constants ===")
	fmt.Printf("ModeRead: %b (%d)\n", ModeRead, ModeRead)
	fmt.Printf("ModeWrite: %b (%d)\n", ModeWrite, ModeWrite)
	fmt.Printf("ModeExecute: %b (%d)\n", ModeExecute, ModeExecute)
	fmt.Printf("Read+Write: %b (%d)\n", ModeRead|ModeWrite, ModeRead|ModeWrite)
}
