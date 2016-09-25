package main

import (
	"sync"
)
type ProcessManager struct {
        procs map[string]*Process
        lock  sync.Mutex
}

func newProcessManager() *ProcessManager {
        procMgr := &ProcessManager{}
        procMgr.procs = make(map[string]*Process)
        return procMgr
}

func (pm *ProcessManager) CreateProcess( config* ConfigEntry ) *Process {
        pm.lock.Lock()
        defer pm.lock.Unlock()

        proc, ok := pm.procs[config.Name]

        if !ok {
                proc = NewProcess( config )
                pm.procs[config.Name] = proc
        }
        return proc
}

func (pm *ProcessManager) Add(name string, proc *Process) {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        pm.procs[name] = proc
}

func (pm *ProcessManager) Remove(name string) *Process {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        proc, _ := pm.procs[name]
        delete(pm.procs, name)
        return proc
}

// return process if found or nil if not found
func (pm *ProcessManager) Find(name string) *Process {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        proc, _ := pm.procs[name]
        return proc
}

// clear all the processes
func (pm *ProcessManager) Clear() {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        pm.procs = make(map[string]*Process)
}

func (pm *ProcessManager) ForEachProcess( procFunc func (p *Process) ) {
        pm.lock.Lock()
        defer pm.lock.Unlock()

        for _, v := range pm.procs {
                procFunc( v )
        }
}

