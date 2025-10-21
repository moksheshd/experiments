package constants

import (
	"testing"
)

func TestConstants(t *testing.T) {
	if PI != 3.14159 {
		t.Errorf("PI = %f, want 3.14159", PI)
	}

	if BEEF != "meat" {
		t.Errorf("BEEF = %s, want meat", BEEF)
	}

	if TWO != 2 {
		t.Errorf("TWO = %d, want 2", TWO)
	}
}

func TestWeekdayConstants(t *testing.T) {
	tests := []struct {
		name  string
		day   int
		value int
	}{
		{"Monday", MONDAY, 1},
		{"Tuesday", TUESDAY, 2},
		{"Wednesday", WEDNESDAY, 3},
		{"Thursday", THURSDAY, 4},
		{"Friday", FRIDAY, 5},
		{"Saturday", SATURDAY, 6},
		{"Sunday", SUNDAY, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.day != tt.value {
				t.Errorf("%s = %d, want %d", tt.name, tt.day, tt.value)
			}
		})
	}
}

func TestGenderEnum(t *testing.T) {
	if UNKNOWN != 0 {
		t.Errorf("UNKNOWN = %d, want 0", UNKNOWN)
	}
	if FEMALE != 1 {
		t.Errorf("FEMALE = %d, want 1", FEMALE)
	}
	if MALE != 2 {
		t.Errorf("MALE = %d, want 2", MALE)
	}
}

func TestGetGenderString(t *testing.T) {
	tests := []struct {
		name   string
		gender Gender
		want   string
	}{
		{"Unknown gender", UNKNOWN, "Unknown"},
		{"Female gender", FEMALE, "Female"},
		{"Male gender", MALE, "Male"},
		{"Invalid gender", Gender(99), "Invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetGenderString(tt.gender); got != tt.want {
				t.Errorf("GetGenderString() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFileModes(t *testing.T) {
	if ModeRead != 1 {
		t.Errorf("ModeRead = %d, want 1", ModeRead)
	}
	if ModeWrite != 2 {
		t.Errorf("ModeWrite = %d, want 2", ModeWrite)
	}
	if ModeExecute != 4 {
		t.Errorf("ModeExecute = %d, want 4", ModeExecute)
	}

	// Test bitwise operations
	readWrite := ModeRead | ModeWrite
	if readWrite != 3 {
		t.Errorf("ModeRead | ModeWrite = %d, want 3", readWrite)
	}
}

func TestDemoFunctions(t *testing.T) {
	// These tests just ensure demo functions don't panic
	t.Run("DemoBasicConstants", func(t *testing.T) {
		DemoBasicConstants()
	})

	t.Run("DemoWeekdayConstants", func(t *testing.T) {
		DemoWeekdayConstants()
	})

	t.Run("DemoTypedConstants", func(t *testing.T) {
		DemoTypedConstants()
	})

	t.Run("DemoGenderEnum", func(t *testing.T) {
		DemoGenderEnum()
	})

	t.Run("DemoFunctionConstants", func(t *testing.T) {
		DemoFunctionConstants()
	})

	t.Run("DemoIotaAdvanced", func(t *testing.T) {
		DemoIotaAdvanced()
	})

	t.Run("DemoFileModes", func(t *testing.T) {
		DemoFileModes()
	})
}

func ExampleGetGenderString() {
	fmt := GetGenderString(FEMALE)
	println(fmt)
	// Output is not checked
}

func ExampleDemoBasicConstants() {
	DemoBasicConstants()
	// Output is printed but not checked
}
