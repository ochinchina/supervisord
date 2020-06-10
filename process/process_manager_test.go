package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stuartcarnie/gopm/config"
)

func TestProcessMgrAdd(t *testing.T) {
	var procs = NewManager()
	procs.Clear()
	procs.Add("test1", NewProcess("github.com/stuartcarnie/gopm", &config.Process{Name: "test1", Group: "test"}))
	assert.NotNil(t, procs.Find("test1"))
}

func TestProcMgrRemove(t *testing.T) {
	var procs = NewManager()
	procs.Clear()
	procs.Add("test1", &Process{})
	proc := procs.Remove("test1")
	assert.NotNil(t, proc, "Failed to remove process")
	proc = procs.Remove("test1")
	assert.Nil(t, proc, "Unexpected value")
}
