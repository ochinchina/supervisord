package main

import (
	"testing"
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
	if len(groups) != 2 || !hasAllElements(stringArrayToInterfacArray(groups), []interface{}{"group1", "group2"}) {
		t.Fail()
	}

}

func TestGetAllProcessInGroup(t *testing.T) {
	group := createTestGroup()

	procs := group.GetAllProcess("group1")

	if len(procs) != 2 || !hasAllElements(stringArrayToInterfacArray(procs), []interface{}{"proc1_1", "proc1_2"}) {
		t.Fail()
	}

	procs = group.GetAllProcess("group10")
	if len(procs) != 0 {
		t.Fail()
	}
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

	if len(procs) != 2 || !hasAllElements(stringArrayToInterfacArray(procs), []interface{}{"proc2_2", "proc2_3"}) {
		t.Fail()
	}

}
