package model_test

import (
	"testing"

	"github.com/stuartcarnie/gopm/model"
)

//
// check if program1 is before the program2 in the Entry
//
func isProgramBefore(t *testing.T, entries []*model.Program, program1, program2 string) bool {
	t.Helper()
	order := 0
	program1Order := -1
	program2Order := -1

	for _, entry := range entries {
		if entry.IsProgram() {
			if entry.Name == program1 {
				program1Order = order
			} else if entry.Name == program2 {
				program2Order = order
			}
			order++
		}
	}

	before := program1Order >= 0 && program1Order < program2Order

	if !before {
		t.Logf("%s Order=%d, %s Order=%d\n", program1, program1Order, program2, program2Order)
	}

	return before
}

func TestProcessSorter_SortProgram(t *testing.T) {
	programs := make([]*model.Program, 0)
	program := new(model.Program)
	program.Name = "prog-1"
	program.DependsOn = []string{"prog-3"}

	programs = append(programs, program)

	program = new(model.Program)
	program.Name = "prog-2"
	program.DependsOn = []string{"prog-1"}

	programs = append(programs, program)

	program = new(model.Program)
	program.Name = "prog-3"
	program.DependsOn = []string{"prog-4", "prog-5"}

	programs = append(programs, program)

	program = new(model.Program)
	program.Name = "prog-5"

	programs = append(programs, program)

	program = new(model.Program)
	program.Name = "prog-4"

	programs = append(programs, program)

	program = new(model.Program)
	program.Name = "prog-6"
	program.Priority = 100

	programs = append(programs, program)

	program = new(model.Program)
	program.Name = "prog-7"
	program.Priority = 99

	programs = append(programs, program)

	result := model.NewProcessSorter().SortProgram(programs)
	if !isProgramBefore(t, result, "prog-5", "prog-3") ||
		!isProgramBefore(t, result, "prog-3", "prog-1") ||
		!isProgramBefore(t, result, "prog-1", "prog-2") ||
		!isProgramBefore(t, result, "prog-7", "prog-6") {
		t.Error("Program sort is incorrect")
	}
}
