package config

import (
	"sort"
	"strings"
)

type ProcessByPriority []*Process

func (p ProcessByPriority) Len() int {
	return len(p)
}

func (p ProcessByPriority) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ProcessByPriority) Less(i, j int) bool {
	return p[i].Priority < p[j].Priority
}

// ProcessSorter sort the program by its priority
type ProcessSorter struct {
	dependsOnGraph      map[string][]string
	procsWithoutDepends []*Process
}

// NewProcessSorter create a sorter
func NewProcessSorter() *ProcessSorter {
	return &ProcessSorter{
		dependsOnGraph:      make(map[string][]string),
		procsWithoutDepends: make([]*Process, 0),
	}
}

func (p *ProcessSorter) initDepends(processes []*Process) {
	// sort by dependsOn
	for _, proc := range processes {
		if len(proc.DependsOn) > 0 {
			dependsOn := proc.DependsOn
			procName := proc.Name
			for _, dependsOnProc := range dependsOn {
				dependsOnProc = strings.TrimSpace(dependsOnProc)
				if dependsOnProc != "" {
					if _, ok := p.dependsOnGraph[procName]; !ok {
						p.dependsOnGraph[procName] = make([]string, 0)
					}
					p.dependsOnGraph[procName] = append(p.dependsOnGraph[procName], dependsOnProc)

				}
			}
		}
	}
}

func (p *ProcessSorter) initProcessWithoutDepends(processes []*Process) {
	dependsOnProcesses := p.getDependsOnInfo()
	for _, config := range processes {
		if _, ok := dependsOnProcesses[config.Name]; !ok {
			p.procsWithoutDepends = append(p.procsWithoutDepends, config)
		}
	}
}

func (p *ProcessSorter) getDependsOnInfo() map[string]string {
	dependsOnProcesses := make(map[string]string)

	for k, v := range p.dependsOnGraph {
		dependsOnProcesses[k] = k
		for _, t := range v {
			dependsOnProcesses[t] = t
		}
	}

	return dependsOnProcesses
}

func (p *ProcessSorter) sortDepends() []string {
	finishedProcesses := make(map[string]string)
	procsWithDependsInfo := p.getDependsOnInfo()
	procsStartOrder := make([]string, 0)

	// Find all processes without depends_on
	for name := range procsWithDependsInfo {
		if _, ok := p.dependsOnGraph[name]; !ok {
			finishedProcesses[name] = name
			procsStartOrder = append(procsStartOrder, name)
		}
	}

	for len(finishedProcesses) < len(procsWithDependsInfo) {
		for progName := range p.dependsOnGraph {
			if _, ok := finishedProcesses[progName]; !ok && p.inFinishedProcess(progName, finishedProcesses) {
				finishedProcesses[progName] = progName
				procsStartOrder = append(procsStartOrder, progName)
			}
		}
	}

	return procsStartOrder
}

func (p *ProcessSorter) inFinishedProcess(processName string, finishedPrograms map[string]string) bool {
	if dependsOn, ok := p.dependsOnGraph[processName]; ok {
		for _, dependProgram := range dependsOn {
			if _, finished := finishedPrograms[dependProgram]; !finished {
				return false
			}
		}
	}
	return true
}

// Sort the processes and return the result
func (p *ProcessSorter) Sort(processes []*Process) []*Process {
	p.initDepends(processes)
	p.initProcessWithoutDepends(processes)
	result := make([]*Process, 0)

	for _, proc := range p.sortDepends() {
		for _, config := range processes {
			if config.Name == proc {
				result = append(result, config)
			}
		}
	}

	sort.Sort(ProcessByPriority(p.procsWithoutDepends))
	for _, p := range p.procsWithoutDepends {
		result = append(result, p)
	}
	return result
}
