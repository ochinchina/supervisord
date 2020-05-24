package process

import (
	"fmt"
	"strings"
	"sync"

	"github.com/stuartcarnie/gopm/model"

	"go.uber.org/zap"
)

// Manager manage all the process in the supervisor
type Manager struct {
	procs map[string]*Process
	lock  sync.Mutex
}

// NewManager create a new Manager object
func NewManager() *Manager {
	return &Manager{
		procs: make(map[string]*Process),
	}
}

// CreateProcess create a process and adds to the manager
func (pm *Manager) CreateProcess(supervisorID string, program *model.Program) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	return pm.createProgram(supervisorID, program)
}

// StartAutoStartPrograms start all the program if its autostart is true
func (pm *Manager) StartAutoStartPrograms() {
	pm.ForEachProcess(func(proc *Process) {
		if proc.program.AutoStart {
			proc.Start(false)
		}
	})
}

func (pm *Manager) createProgram(supervisorID string, program *model.Program) *Process {
	proc, ok := pm.procs[program.Name]
	if !ok {
		proc = NewProcess(supervisorID, program)
		pm.procs[program.Name] = proc
	}
	zap.L().Info("Created program", zap.String("program", program.Name))
	return proc
}

// Add add the process to this process manager
func (pm *Manager) Add(name string, proc *Process) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs[name] = proc
}

// Remove remove the process from the manager
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
	zap.L().Info("remove process", zap.String("name", name))
	return proc
}

// Find find process by program name return process if found or nil if not found
func (pm *Manager) Find(name string) *Process {
	procs := pm.FindMatch(name)
	if len(procs) == 1 {
		if procs[0].Name() == name || name == fmt.Sprintf("%s:%s", procs[0].Group(), procs[0].Name()) {
			return procs[0]
		}
	}
	return nil
}

// FindMatch find the program with one of following format:
// - group.program
// - group.*
// - program
func (pm *Manager) FindMatch(name string) []*Process {
	result := make([]*Process, 0)
	if pos := strings.Index(name, "."); pos != -1 {
		groupName := name[0:pos]
		programName := name[pos+1:]
		pm.ForEachProcess(func(p *Process) {
			if p.Group() == groupName {
				if programName == "*" || programName == p.Name() {
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
		zap.L().Debug("Failed to find process", zap.String("name", name))
	}
	return result
}

// Clear clear all the processes
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

// StopAllProcesses stop all the processes managed by this manager
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
	progConfigs := make([]*model.Program, 0)
	for _, proc := range procs {
		progConfigs = append(progConfigs, proc.program)
	}

	result := make([]*Process, 0)
	p := model.NewProcessSorter()
	for _, program := range p.SortProgram(progConfigs) {
		for _, proc := range procs {
			if proc.program == program {
				result = append(result, proc)
			}
		}
	}

	return result
}
