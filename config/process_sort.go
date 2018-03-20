package config

import (
	"sort"
	"strings"
)

type ProgramByPriority []*ConfigEntry

func (p ProgramByPriority) Len() int {
	return len(p)
}

func (p ProgramByPriority) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ProgramByPriority) Less(i, j int) bool {
	return p[i].GetInt("priority", 999) < p[j].GetInt("priority", 999)
}

type ProcessSorter struct {
	depends_on_gragh      map[string][]string
	procs_without_depends []*ConfigEntry
}

func NewProcessSorter() *ProcessSorter {
	return &ProcessSorter{depends_on_gragh: make(map[string][]string),
		procs_without_depends: make([]*ConfigEntry, 0)}
}

func (p *ProcessSorter) initDepends(program_configs []*ConfigEntry) {
	//sort by depends_on
	for _, config := range program_configs {
		if config.IsProgram() && config.HasParameter("depends_on") {
			depends_on := config.GetString("depends_on", "")
			prog_name := config.GetProgramName()
			for _, depends_on_prog := range strings.Split(depends_on, ",") {
				depends_on_prog = strings.TrimSpace(depends_on_prog)
				if depends_on_prog != "" {
					if _, ok := p.depends_on_gragh[prog_name]; !ok {
						p.depends_on_gragh[prog_name] = make([]string, 0)
					}
					p.depends_on_gragh[prog_name] = append(p.depends_on_gragh[prog_name], depends_on_prog)

				}
			}
		}
	}

}

func (p *ProcessSorter) initProgramWithoutDepends(program_configs []*ConfigEntry) {
	depends_on_programs := p.getDependsOnInfo()
	for _, config := range program_configs {
		if config.IsProgram() {
			if _, ok := depends_on_programs[config.GetProgramName()]; !ok {
				p.procs_without_depends = append(p.procs_without_depends, config)
			}
		}
	}
}

func (p *ProcessSorter) getDependsOnInfo() map[string]string {
	depends_on_programs := make(map[string]string)

	for k, v := range p.depends_on_gragh {
		depends_on_programs[k] = k
		for _, t := range v {
			depends_on_programs[t] = t
		}
	}

	return depends_on_programs
}

func (p *ProcessSorter) sortDepends() []string {
	finished_programs := make(map[string]string)
	progs_with_depends_info := p.getDependsOnInfo()
	progs_start_order := make([]string, 0)

	//get all process without depends
	for prog_name := range progs_with_depends_info {
		if _, ok := p.depends_on_gragh[prog_name]; !ok {
			finished_programs[prog_name] = prog_name
			progs_start_order = append(progs_start_order, prog_name)
		}
	}

	for len(finished_programs) < len(progs_with_depends_info) {
		for prog_name := range p.depends_on_gragh {
			if _, ok := finished_programs[prog_name]; !ok && p.inFinishedPrograms(prog_name, finished_programs) {
				finished_programs[prog_name] = prog_name
				progs_start_order = append(progs_start_order, prog_name)
			}
		}
	}

	return progs_start_order
}

func (p *ProcessSorter) inFinishedPrograms(program_name string, finished_programs map[string]string) bool {
	if depends_on, ok := p.depends_on_gragh[program_name]; ok {
		for _, depend_program := range depends_on {
			if _, finished := finished_programs[depend_program]; !finished {
				return false
			}
		}
	}
	return true
}

/*func (p *ProcessSorter) SortProcess(procs []*Process) []*Process {
	prog_configs := make([]*ConfigEntry, 0)
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

func (p *ProcessSorter) SortProgram(program_configs []*ConfigEntry) []*ConfigEntry {
	p.initDepends(program_configs)
	p.initProgramWithoutDepends(program_configs)
	result := make([]*ConfigEntry, 0)

	for _, prog := range p.sortDepends() {
		for _, config := range program_configs {
			if config.IsProgram() && config.GetProgramName() == prog {
				result = append(result, config)
			}
		}
	}

	sort.Sort(ProgramByPriority(p.procs_without_depends))
	for _, p := range p.procs_without_depends {
		result = append(result, p)
	}
	return result
}

/*func sortProcess(procs []*Process) []*Process {
	return NewProcessSorter().SortProcess(procs)
}*/

func sortProgram(configs []*ConfigEntry) []*ConfigEntry {
	return NewProcessSorter().SortProgram(configs)
}
