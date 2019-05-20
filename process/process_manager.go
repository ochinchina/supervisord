package process

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ochinchina/supervisord/config"
	log "github.com/sirupsen/logrus"
)

type ProcessManager struct {
	procs          map[string]*Process
	eventListeners map[string]*Process
	lock           sync.Mutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{procs: make(map[string]*Process),
		eventListeners: make(map[string]*Process),
	}
}

func (pm *ProcessManager) CreateProcess(supervisor_id string, config *config.ConfigEntry) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	if config.IsProgram() {
		return pm.createProgram(supervisor_id, config)
	} else if config.IsEventListener() {
		return pm.createEventListener(supervisor_id, config)
	} else {
		return nil
	}
}

func (pm *ProcessManager) StartAutoStartPrograms() {
	pm.ForEachProcess(func(proc *Process) {
		if proc.isAutoStart() {
			proc.Start(false)
		}
	})
}

func (pm *ProcessManager) createProgram(supervisor_id string, config *config.ConfigEntry) *Process {
	procName := config.GetProgramName()

	proc, ok := pm.procs[procName]

	if !ok {
		proc = NewProcess(supervisor_id, config)
		pm.procs[procName] = proc
	}
	log.Info("create process:", procName)
	return proc
}

func (pm *ProcessManager) createEventListener(supervisor_id string, config *config.ConfigEntry) *Process {
	eventListenerName := config.GetEventListenerName()

	evtListener, ok := pm.eventListeners[eventListenerName]

	if !ok {
		evtListener = NewProcess(supervisor_id, config)
		pm.eventListeners[eventListenerName] = evtListener
	}
	log.Info("create event listener:", eventListenerName)
	return evtListener
}

func (pm *ProcessManager) Add(name string, proc *Process) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs[name] = proc
	log.Info("add process:", name)
}

// remove the process from the manager
//
// Arguments:
// name - the name of program
//
// Return the process or nil
func (pm *ProcessManager) Remove(name string) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	proc, _ := pm.procs[name]
	delete(pm.procs, name)
	log.Info("remove process:", name)
	return proc
}

// return process if found or nil if not found
func (pm *ProcessManager) Find(name string) *Process {
	procs := pm.FindMatch(name)
	if len(procs) == 1 {
		if procs[0].GetName() == name || name == fmt.Sprintf("%s:%s", procs[0].GetGroup(), procs[0].GetName()) {
			return procs[0]
		}
	}
	return nil
}

func (pm *ProcessManager) FindMatch(name string) []*Process {
	result := make([]*Process, 0)
	if pos := strings.Index(name, ":"); pos != -1 {
		groupName := name[0:pos]
		programName := name[pos+1:]
		pm.ForEachProcess(func(p *Process) {
			if p.GetGroup() == groupName {
				if programName == "*" || programName == p.GetName() {
					result = append(result, p)
				}
			}
		})
	} else {
		pm.lock.Lock()
		defer pm.lock.Unlock()
		proc, ok := pm.procs[name]
		if ok {
			result = append(result, proc)
		}
	}
	if len(result) <= 0 {
		log.Info("fail to find process:", name)
	}
	return result
}

// clear all the processes
func (pm *ProcessManager) Clear() {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs = make(map[string]*Process)
}

// process each process in sync mode
func (pm *ProcessManager) ForEachProcess(procFunc func(p *Process)) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	procs := pm.getAllProcess()
	for _, proc := range procs {
		procFunc(proc)
	}
}

// handle each process in async mode
// Args:
// - procFunc, the function to handle the process
// - done, signal the process is completed
// Returns: number of total processes
func (pm *ProcessManager) AsyncForEachProcess(procFunc func(p *Process), done chan *Process) int {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	procs := pm.getAllProcess()

	for _, proc := range procs {
		go forOneProcess(proc, procFunc, done)
	}
	return len(procs)
}

func forOneProcess(proc *Process, action func(p *Process), done chan *Process) {
	action(proc)
	done <- proc
}

func (pm *ProcessManager) getAllProcess() []*Process {
	tmpProcs := make([]*Process, 0)
	for _, proc := range pm.procs {
		tmpProcs = append(tmpProcs, proc)
	}
	return sortProcess(tmpProcs)
}

func (pm *ProcessManager) StopAllProcesses() {
	var wg sync.WaitGroup

	pm.ForEachProcess(func(proc *Process) {
		wg.Add(1)

		go func(wg *sync.WaitGroup) {
			defer wg.Done()

			proc.Stop(true)
		}(&wg)
	})

	wg.Wait()
}

func sortProcess(procs []*Process) []*Process {
	prog_configs := make([]*config.ConfigEntry, 0)
	for _, proc := range procs {
		if proc.config.IsProgram() {
			prog_configs = append(prog_configs, proc.config)
		}
	}

	result := make([]*Process, 0)
	p := config.NewProcessSorter()
	for _, config := range p.SortProgram(prog_configs) {
		for _, proc := range procs {
			if proc.config == config {
				result = append(result, proc)
			}
		}
	}

	return result
}
