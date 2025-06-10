package process

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ochinchina/supervisord/config"
	log "github.com/sirupsen/logrus"
)

// Manager manage all the process in the supervisor
type Manager struct {
	procs          map[string]*Process
	eventListeners map[string]*Process
	lock           sync.Mutex
}

// NewManager creates new Manager object
func NewManager() *Manager {
	return &Manager{procs: make(map[string]*Process),
		eventListeners: make(map[string]*Process),
	}
}

// CreateProcess creates process (program or event listener) and adds to Manager object
func (pm *Manager) CreateProcess(supervisorID string, config *config.Entry) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	if config.IsProgram() {
		return pm.createProgram(supervisorID, config)
	} else if config.IsEventListener() {
		return pm.createEventListener(supervisorID, config)
	} else {
		return nil
	}
}

// StartAutoStartPrograms starts all programs that set as should be autostarted
func (pm *Manager) StartAutoStartPrograms() {
	pm.ForEachProcess(func(proc *Process) {
		if proc.isAutoStart() {
			proc.Start(false)
		}
	})
}

func (pm *Manager) createProgram(supervisorID string, config *config.Entry) *Process {
	procName := config.GetProgramName()

	proc, ok := pm.procs[procName]

	if !ok {
		proc = NewProcess(supervisorID, config)
		pm.procs[procName] = proc
	}
	log.Info("create process:", procName)
	return proc
}

func (pm *Manager) createEventListener(supervisorID string, config *config.Entry) *Process {
	eventListenerName := config.GetEventListenerName()

	evtListener, ok := pm.eventListeners[eventListenerName]

	if !ok {
		evtListener = NewProcess(supervisorID, config)
		pm.eventListeners[eventListenerName] = evtListener
	}
	log.Info("create event listener:", eventListenerName)
	return evtListener
}

// Add process to Manager object
func (pm *Manager) Add(name string, proc *Process) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs[name] = proc
	log.Info("add process:", name)
}

// Remove process from Manager object
//
// Arguments:
// name - the name of program
//
// Return the process or nil
func (pm *Manager) Remove(name string) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	proc, _ := pm.procs[name]
	delete(pm.procs, name)
	log.Info("remove process:", name)
	return proc
}

// Find process by program name. Returns process or nil if process is not listed in Manager object
func (pm *Manager) Find(name string) *Process {
	procs := pm.FindMatch(name)
	if len(procs) == 1 {
		if procs[0].GetName() == name || name == fmt.Sprintf("%s:%s", procs[0].GetGroup(), procs[0].GetName()) {
			return procs[0]
		}
	}
	return nil
}

// FindMatch lookup program with one of following format:
// - group:program
// - group:*
// - program
func (pm *Manager) FindMatch(name string) []*Process {
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

// Clear all the processes from Manager object
func (pm *Manager) Clear() {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs = make(map[string]*Process)
}

// ForEachProcess process each process in sync mode
func (pm *Manager) ForEachProcess(procFunc func(p *Process)) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	procs := pm.getAllProcess()
	for _, proc := range procs {
		procFunc(proc)
	}
}

// AsyncForEachProcess handle each process in async mode
// Args:
// - procFunc, the function to handle the process
// - done, signal the process is completed
// Returns: number of total processes
func (pm *Manager) AsyncForEachProcess(procFunc func(p *Process), done chan *Process) int {
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

func (pm *Manager) getAllProcess() []*Process {
	tmpProcs := make([]*Process, 0)
	for _, proc := range pm.procs {
		tmpProcs = append(tmpProcs, proc)
	}
	return sortProcess(tmpProcs)
}

// StopAllProcesses stop all the processes listed in Manager object
func (pm *Manager) StopAllProcesses() {
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
	progConfigs := make([]*config.Entry, 0)
	for _, proc := range procs {
		if proc.config.IsProgram() {
			progConfigs = append(progConfigs, proc.config)
		}
	}

	result := make([]*Process, 0)
	p := config.NewProcessSorter()
	for _, config := range p.SortProgram(progConfigs) {
		for _, proc := range procs {
			if proc.config == config {
				result = append(result, proc)
			}
		}
	}

	return result
}
