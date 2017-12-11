package process

import (
	"testing"
)

var procs *ProcessManager = NewProcessManager()

func TestProcessMgrAdd(t *testing.T) {
	procs.Clear()
	procs.Add("test1", &Process{})

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
