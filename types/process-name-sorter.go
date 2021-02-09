package types

import (
	"sort"
	"strings"
)

// ProcessNameSorter sort the process info by program name
type ProcessNameSorter struct {
	processes []ProcessInfo
}

// NewwProcessNameSorter create a new ProcessNameSorter object
func NewwProcessNameSorter(processes []ProcessInfo) *ProcessNameSorter {
	return &ProcessNameSorter{processes: processes}
}

// Len return the number of programs
func (pns *ProcessNameSorter) Len() int {
	return len(pns.processes)
}

// Less return true if the program name of ith process is less than the program name of jth process
func (pns *ProcessNameSorter) Less(i, j int) bool {
	return strings.Compare(pns.processes[i].Name, pns.processes[j].Name) < 0
}

// Swap swap the ith program and jth program
func (pns *ProcessNameSorter) Swap(i, j int) {
	info := pns.processes[i]
	pns.processes[i] = pns.processes[j]
	pns.processes[j] = info
}

// SortProcessInfos sort the process information with program name
func SortProcessInfos(processes []ProcessInfo) {
	sorter := NewwProcessNameSorter(processes)
	sort.Sort(sorter)
}
