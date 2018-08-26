package types

import (
	"sort"
	"strings"
)

type ProcessNameSorter struct {
	processes []ProcessInfo
}

func NewwProcessNameSorter(processes []ProcessInfo) *ProcessNameSorter {
	return &ProcessNameSorter{processes: processes}
}

func (pns *ProcessNameSorter) Len() int {
	return len(pns.processes)
}

func (pns *ProcessNameSorter) Less(i, j int) bool {
	return strings.Compare(pns.processes[i].Name, pns.processes[j].Name) < 0
}

func (pns *ProcessNameSorter) Swap(i, j int) {
	info := pns.processes[i]
	pns.processes[i] = pns.processes[j]
	pns.processes[j] = info
}

func SortProcessInfos(processes []ProcessInfo) {
	sorter := NewwProcessNameSorter(processes)
	sort.Sort(sorter)
}
