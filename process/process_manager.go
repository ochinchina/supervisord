package process

import (
	"fmt"
	"strings"
	"sync"

	"github.com/stuartcarnie/gopm/config"
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

// CreateOrUpdateProcess creates a new process and adds it to the manager or updates an existing process.
func (pm *Manager) CreateOrUpdateProcess(supervisorID string, after *config.Process) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	proc, ok := pm.procs[after.Name]
	if !ok {
		proc = NewProcess(supervisorID, after)
		pm.procs[after.Name] = proc
		zap.L().Info("Created program", zap.String("program", after.Name))
	} else {
		proc.UpdateConfig(after)
		zap.L().Info("Updated program", zap.String("program", after.Name))
	}

	return proc
}

// StartAutoStartPrograms start all the program if its autostart is true
func (pm *Manager) StartAutoStartPrograms() {
	pm.ForEachProcess(func(proc *Process) {
		if proc.config.AutoStart {
			proc.Start(false)
		}
	})
}

func (pm *Manager) createProcess(supervisorID string, process *config.Process) *Process {
	// TODO(sgc): Update existing programs; e.g. cron schedule, etc
	proc, ok := pm.procs[process.Name]
	if !ok {
		proc = NewProcess(supervisorID, process)
		pm.procs[process.Name] = proc
	}
	zap.L().Info("Created program", zap.String("program", process.Name))
	return proc
}

// Add add the process to this process manager
func (pm *Manager) Add(name string, proc *Process) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs[name] = proc
}

// Remove remove the process from the manager
// Return the removed process or nil
func (pm *Manager) Remove(name string) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	proc, _ := pm.procs[name]
	delete(pm.procs, name)
	zap.L().Info("Removed process", zap.String("name", name))
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
	pm.lock.Lock()
	defer pm.lock.Unlock()

	processes := pm.getAllProcess()
	var wg sync.WaitGroup
	wg.Add(len(processes))
	for _, p := range processes {
		go func(proc *Process) {
			defer wg.Done()
			proc.Stop(true)
		}(p)
	}

	wg.Wait()
}

func sortProcess(procs []*Process) []*Process {
	progConfigs := make([]*config.Process, 0)
	for _, proc := range procs {
		progConfigs = append(progConfigs, proc.config)
	}

	result := make([]*Process, 0)
	p := config.NewProcessSorter()
	for _, program := range p.Sort(progConfigs) {
		for _, proc := range procs {
			if proc.config == program {
				result = append(result, proc)
			}
		}
	}

	return result
}
