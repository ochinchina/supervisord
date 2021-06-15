package config

import (
	"sort"
	"strings"
)

// ProgramByPriority sort program by its priority
type ProgramByPriority []*Entry

// Len returns amount of programs
func (p ProgramByPriority) Len() int {
	return len(p)
}

// Swap swaps program i and program j
func (p ProgramByPriority) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// Less returns true if the priority i-th program is less than the priority of j-th program
func (p ProgramByPriority) Less(i, j int) bool {
	return p[i].GetInt("priority", 999) < p[j].GetInt("priority", 999)
}

// ProcessSorter sort the program by its priority
type ProcessSorter struct {
	dependsOnGraph       map[string][]string
	procsWithooutDepends []*Entry
}

// NewProcessSorter creates sorter
func NewProcessSorter() *ProcessSorter {
	return &ProcessSorter{dependsOnGraph: make(map[string][]string),
		procsWithooutDepends: make([]*Entry, 0)}
}

func (p *ProcessSorter) initDepends(programConfigs []*Entry) {
	// sort by dependsOn
	for _, config := range programConfigs {
		if config.IsProgram() && config.HasParameter("depends_on") {
			dependsOn := config.GetString("depends_on", "")
			progName := config.GetProgramName()
			for _, dependsOnProg := range strings.Split(dependsOn, ",") {
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

func (p *ProcessSorter) initProgramWithoutDepends(programConfigs []*Entry) {
	dependsOnPrograms := p.getDependsOnInfo()
	for _, config := range programConfigs {
		if config.IsProgram() {
			if _, ok := dependsOnPrograms[config.GetProgramName()]; !ok {
				p.procsWithooutDepends = append(p.procsWithooutDepends, config)
			}
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

/*func (p *ProcessSorter) SortProcess(procs []*Process) []*Process {
	prog_configs := make([]*Entry, 0)
	for _, proc := range procs {
		if proc.config.IsProgram() {
			prog_configs = append(prog_configs, proc.config)
		}
	}

	result := make([]*Process, 0)
	for _, config := range p.SortProgram(prog_configs) {
		for _, proc := range procs {
			if proc.config == config {
				result = append(result, proc)
			}
		}
	}

	return result
}*/

// SortProgram sort the program  and return the result
func (p *ProcessSorter) SortProgram(programConfigs []*Entry) []*Entry {
	p.initDepends(programConfigs)
	p.initProgramWithoutDepends(programConfigs)
	result := make([]*Entry, 0)

	for _, prog := range p.sortDepends() {
		for _, config := range programConfigs {
			if config.IsProgram() && config.GetProgramName() == prog {
				result = append(result, config)
			}
		}
	}

	sort.Sort(ProgramByPriority(p.procsWithooutDepends))
	for _, p := range p.procsWithooutDepends {
		result = append(result, p)
	}
	return result
}

/*func sortProcess(procs []*Process) []*Process {
	return NewProcessSorter().SortProcess(procs)
}*/

func sortProgram(configs []*Entry) []*Entry {
	return NewProcessSorter().SortProgram(configs)
}
