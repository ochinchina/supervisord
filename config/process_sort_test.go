package config

import (
	"fmt"
	"testing"
)

//
// check if program_1 is before the program_2 in the ConfigEntry
//
func isProgramBefore(entries []*ConfigEntry, program_1 string, program_2 string) bool {
	order := 0
	program_1_order := -1
	program_2_order := -1

	for _, entry := range entries {
		if entry.IsProgram() {
			if entry.GetProgramName() == program_1 {
				program_1_order = order
			} else if entry.GetProgramName() == program_2 {
				program_2_order = order
			}
			order++
		}
	}

	fmt.Printf("program_1_order=%d, program_2_order=%d\n", program_1_order, program_2_order)

	return program_1_order >= 0 && program_1_order < program_2_order
}
func TestSortProgram(t *testing.T) {
	entries := make([]*ConfigEntry, 0)
	entry := NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-1"
	entry.keyValues["depends_on"] = "prog-3"

	entries = append(entries, entry)

	entry = NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-2"
	entry.keyValues["depends_on"] = "prog-1"

	entries = append(entries, entry)

	entry = NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-3"
	entry.keyValues["depends_on"] = "prog-4,prog-5"

	entries = append(entries, entry)

	entry = NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-5"

	entries = append(entries, entry)

	entry = NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-4"

	entries = append(entries, entry)

	entry = NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-6"
	entry.keyValues["priority"] = "100"

	entries = append(entries, entry)

	entry = NewConfigEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-7"
	entry.keyValues["priority"] = "99"

	entries = append(entries, entry)

	result := sortProgram(entries)
	for _, e := range result {
		fmt.Printf("%s\n", e.GetProgramName())
	}

	if	!isProgramBefore(result, "prog-5", "prog-3") ||
		!isProgramBefore(result, "prog-3", "prog-1") ||
		!isProgramBefore(result, "prog-1", "prog-2") ||
		!isProgramBefore(result, "prog-7", "prog-6") {
		t.Error("Program sort is incorrect")
	}

}
