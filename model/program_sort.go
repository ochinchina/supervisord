package model

import (
	"sort"
	"strings"
)

type ProgramByPriority []*Program

func (p ProgramByPriority) Len() int {
	return len(p)
}

func (p ProgramByPriority) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ProgramByPriority) Less(i, j int) bool {
	return p[i].Priority < p[j].Priority
}

// ProcessSorter sort the program by its priority
type ProcessSorter struct {
	dependsOnGraph      map[string][]string
	procsWithoutDepends []*Program
}

// NewProcessSorter create a sorter
func NewProcessSorter() *ProcessSorter {
	return &ProcessSorter{
		dependsOnGraph:      make(map[string][]string),
		procsWithoutDepends: make([]*Program, 0),
	}
}

func (p *ProcessSorter) initDepends(programs []*Program) {
	// sort by dependsOn
	for _, program := range programs {
		if len(program.DependsOn) > 0 {
			dependsOn := program.DependsOn
			progName := program.Name
			for _, dependsOnProg := range dependsOn {
				dependsOnProg = strings.TrimSpace(dependsOnProg)
				if dependsOnProg != "" {
					if _, ok := p.dependsOnGraph[progName]; !ok {
						p.dependsOnGraph[progName] = make([]string, 0)
					}
					p.dependsOnGraph[progName] = append(p.dependsOnGraph[progName], dependsOnProg)

				}
			}
		}
	}
}

func (p *ProcessSorter) initProgramWithoutDepends(programConfigs []*Program) {
	dependsOnPrograms := p.getDependsOnInfo()
	for _, config := range programConfigs {
		if _, ok := dependsOnPrograms[config.Name]; !ok {
			p.procsWithoutDepends = append(p.procsWithoutDepends, config)
		}
	}
}

func (p *ProcessSorter) getDependsOnInfo() map[string]string {
	dependsOnPrograms := make(map[string]string)

	for k, v := range p.dependsOnGraph {
		dependsOnPrograms[k] = k
		for _, t := range v {
			dependsOnPrograms[t] = t
		}
	}

	return dependsOnPrograms
}

func (p *ProcessSorter) sortDepends() []string {
	finishedPrograms := make(map[string]string)
	progsWithDependsInfo := p.getDependsOnInfo()
	progsStartOrder := make([]string, 0)

	// get all process without depends
	for progName := range progsWithDependsInfo {
		if _, ok := p.dependsOnGraph[progName]; !ok {
			finishedPrograms[progName] = progName
			progsStartOrder = append(progsStartOrder, progName)
		}
	}

	for len(finishedPrograms) < len(progsWithDependsInfo) {
		for progName := range p.dependsOnGraph {
			if _, ok := finishedPrograms[progName]; !ok && p.inFinishedPrograms(progName, finishedPrograms) {
				finishedPrograms[progName] = progName
				progsStartOrder = append(progsStartOrder, progName)
			}
		}
	}

	return progsStartOrder
}

func (p *ProcessSorter) inFinishedPrograms(programName string, finishedPrograms map[string]string) bool {
	if dependsOn, ok := p.dependsOnGraph[programName]; ok {
		for _, dependProgram := range dependsOn {
			if _, finished := finishedPrograms[dependProgram]; !finished {
				return false
			}
		}
	}
	return true
}

// SortProgram sort the program  and return the result
func (p *ProcessSorter) SortProgram(programs []*Program) []*Program {
	p.initDepends(programs)
	p.initProgramWithoutDepends(programs)
	result := make([]*Program, 0)

	for _, prog := range p.sortDepends() {
		for _, config := range programs {
			if config.Name == prog {
				result = append(result, config)
			}
		}
	}

	sort.Sort(ProgramByPriority(p.procsWithoutDepends))
	for _, p := range p.procsWithoutDepends {
		result = append(result, p)
	}
	return result
}
