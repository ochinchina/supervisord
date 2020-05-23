package process

import (
	"supervisord/model"
	"testing"
)

var procs = NewManager()

func TestProcessMgrAdd(t *testing.T) {
	procs.Clear()
	procs.Add("test1", NewProcess("supervisord", &model.Program{Name: "test1", Group: "test"}))

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
