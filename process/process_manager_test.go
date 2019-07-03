package process

import (
	"github.com/ochinchina/supervisord/config"
	"testing"
)

var procs *ProcessManager = NewProcessManager()

func TestProcessMgrAdd(t *testing.T) {
	entry := &config.ConfigEntry{ConfigDir: ".", Group: "test", Name: "program:test1"}
	procs.Clear()
	procs.Add("test1", NewProcess("supervisord", entry))

	if procs.Find("test1") == nil {
		t.Error("fail to add process")
	}
}

func TestProcMgrRemove(t *testing.T) {
	procs.Clear()
	procs.Add("test1", &Process{})
	proc := procs.Remove("test1")

	if proc == nil {
		t.Error("fail to remove process")
	}

	proc = procs.Remove("test1")
	if proc != nil {
		t.Error("fail to remove process")
	}
}
