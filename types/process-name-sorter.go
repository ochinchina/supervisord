package types

import (
	"reflect"
	"sort"
)

// ProcessNameSorter sort the process info by program name
type ProcessNameSorter struct {
	processes []ProcessInfo
}

// NewProcessNameSorter creates new ProcessNameSorter object
func NewProcessNameSorter(processes []ProcessInfo) *ProcessNameSorter {
	return &ProcessNameSorter{processes: processes}
}

// Len returns amount of programs
func (pns *ProcessNameSorter) Len() int {
	return len(pns.processes)
}

// Less returns true if program name of i-th process is less than the program name of j-th process
func (pns *ProcessNameSorter) Less(i, j int) bool {
	return pns.processes[i].Name < pns.processes[j].Name
}

// Swap i-th program and j-th program
func (pns *ProcessNameSorter) Swap(i, j int) {
	swapF := reflect.Swapper(pns.processes)
	swapF(i,j)
}

// SortProcessInfos sorts the process information by program name
func SortProcessInfos(processes []ProcessInfo) {
	sorter := NewProcessNameSorter(processes)
	sort.Sort(sorter)
}
