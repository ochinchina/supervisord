package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stuartcarnie/gopm/config"
)

//
// check if program1 is before the program2 in the Entry
//
func isProcessBefore(t *testing.T, entries []*config.Process, process1, process2 string) bool {
	t.Helper()
	order := 0
	process1Order := -1
	process2Order := -1

	for _, entry := range entries {
		if entry.Name == process1 {
			process1Order = order
		} else if entry.Name == process2 {
			process2Order = order
		}
		order++
	}

	before := process1Order >= 0 && process1Order < process2Order

	if !before {
		t.Logf("%s Order=%d, %s Order=%d\n", process1, process1Order, process2, process2Order)
	}

	return before
}

func TestProcessSorter_Sort(t *testing.T) {
	processes := make([]*config.Process, 0)
	program := new(config.Process)
	program.Name = "prog-1"
	program.DependsOn = []string{"prog-3"}

	processes = append(processes, program)

	program = new(config.Process)
	program.Name = "prog-2"
	program.DependsOn = []string{"prog-1"}

	processes = append(processes, program)

	program = new(config.Process)
	program.Name = "prog-3"
	program.DependsOn = []string{"prog-4", "prog-5"}

	processes = append(processes, program)

	program = new(config.Process)
	program.Name = "prog-5"

	processes = append(processes, program)

	program = new(config.Process)
	program.Name = "prog-4"

	processes = append(processes, program)

	program = new(config.Process)
	program.Name = "prog-6"
	program.Priority = 100

	processes = append(processes, program)

	program = new(config.Process)
	program.Name = "prog-7"
	program.Priority = 99

	processes = append(processes, program)

	result := config.NewProcessSorter().Sort(processes)
	assert.True(t, isProcessBefore(t, result, "prog-5", "prog-3"))
	assert.True(t, isProcessBefore(t, result, "prog-3", "prog-1"))
	assert.True(t, isProcessBefore(t, result, "prog-1", "prog-2"))
	assert.True(t, isProcessBefore(t, result, "prog-7", "prog-6"))
}
