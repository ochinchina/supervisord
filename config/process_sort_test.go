package config

import (
	"fmt"
	"testing"
)

//
// check if program1 is before the program2 in the Entry
//
func isProgramBefore(entries []*Entry, program1 string, program2 string) bool {
	order := 0
	program1Order := -1
	program2Order := -1

	for _, entry := range entries {
		if entry.IsProgram() {
			if entry.GetProgramName() == program1 {
				program1Order = order
			} else if entry.GetProgramName() == program2 {
				program2Order = order
			}
			order++
		}
	}

	fmt.Printf("%s Order=%d, %s Order=%d\n", program1, program1Order, program2, program2Order)

	return program1Order >= 0 && program1Order < program2Order
}
func TestSortProgram(t *testing.T) {
	entries := make([]*Entry, 0)
	entry := NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-1"
	entry.keyValues["depends_on"] = "prog-3"

	entries = append(entries, entry)

	entry = NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-2"
	entry.keyValues["depends_on"] = "prog-1"

	entries = append(entries, entry)

	entry = NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-3"
	entry.keyValues["depends_on"] = "prog-4,prog-5"

	entries = append(entries, entry)

	entry = NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-5"

	entries = append(entries, entry)

	entry = NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-4"

	entries = append(entries, entry)

	entry = NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-6"
	entry.keyValues["priority"] = "100"

	entries = append(entries, entry)

	entry = NewEntry(".")
	entry.Group = "group:group-1"
	entry.Name = "program:prog-7"
	entry.keyValues["priority"] = "99"

	entries = append(entries, entry)

	result := sortProgram(entries)
	for _, e := range result {
		fmt.Printf("%s\n", e.GetProgramName())
	}

	if !isProgramBefore(result, "prog-5", "prog-3") ||
		!isProgramBefore(result, "prog-3", "prog-1") ||
		!isProgramBefore(result, "prog-1", "prog-2") ||
		!isProgramBefore(result, "prog-7", "prog-6") {
		t.Error("Program sort is incorrect")
	}

}
