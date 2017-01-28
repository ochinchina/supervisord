package main

import (
	log "github.com/Sirupsen/logrus"
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
	procName := config.Name[len("program:"):]

        proc, ok := pm.procs[procName]

        if !ok {
                proc = NewProcess( config )
                pm.procs[procName] = proc
        } 
	log.Info( "create process:", procName )
        return proc
}

func (pm *ProcessManager) Add(name string, proc *Process) {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        pm.procs[name] = proc
	log.Info( "add process:", name )
}

func (pm *ProcessManager) Remove(name string) *Process {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        proc, _ := pm.procs[name]
        delete(pm.procs, name)
	log.Info( "remove process:", name )
        return proc
}

// return process if found or nil if not found
func (pm *ProcessManager) Find(name string) *Process {
        pm.lock.Lock()
        defer pm.lock.Unlock()
        proc, ok := pm.procs[name]
	if ok {
		log.Debug( "succeed to find process:", name)
	} else {
		log.Info( "fail to find process:", name )
	}
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

