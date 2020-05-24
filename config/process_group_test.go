package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestGroup() *ProcessGroup {
	group := NewProcessGroup()

	group.Add("group1", "proc1_1")
	group.Add("group1", "proc1_2")
	group.Add("group2", "proc2_1")
	group.Add("group2", "proc2_2")
	group.Add("group2", "proc2_3")

	return group
}

func TestGetAllGroup(t *testing.T) {
	group := createTestGroup()

	groups := group.GetAllGroup()
	assert.ElementsMatch(t, groups, []string{"group1", "group2"})
}

func TestGetAllProcessInGroup(t *testing.T) {
	group := createTestGroup()

	procs := group.GetAllProcess("group1")
	assert.ElementsMatch(t, procs, []string{"proc1_1", "proc1_2"})

	procs = group.GetAllProcess("group10")
	assert.Empty(t, procs)
}

func TestInGroup(t *testing.T) {
	group := createTestGroup()

	if !group.InGroup("proc2_2", "group2") || group.InGroup("proc1_1", "group2") {
		t.Fail()
	}
}

func TestRemoveFromGroup(t *testing.T) {
	group := createTestGroup()

	group.Remove("proc2_1")

	procs := group.GetAllProcess("group2")
	assert.ElementsMatch(t, procs, []string{"proc2_2", "proc2_3"})
}

func TestGroupDiff(t *testing.T) {
	group1 := NewProcessGroup()
	group1.Add("group-1", "proc-11")
	group1.Add("group-1", "proc-12")
	group1.Add("group-2", "proc-21")

	group2 := NewProcessGroup()
	group2.Add("group-1", "proc-11")
	group2.Add("group-1", "proc-12")
	group2.Add("group-1", "proc-13")
	group2.Add("group-3", "proc-31")

	added, changed, removed := group2.Sub(group1)
	assert.ElementsMatch(t, []string{"group-3"}, added)
	assert.ElementsMatch(t, []string{"group-1"}, changed)
	assert.ElementsMatch(t, []string{"group-2"}, removed)
}
