package config

import (
	"testing"

	"github.com/ochinchina/supervisord/util"
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
	if len(groups) != 2 || !util.HasAllElements(util.StringArrayToInterfacArray(groups), []interface{}{"group1", "group2"}) {
		t.Fail()
	}

}

func TestGetAllProcessInGroup(t *testing.T) {
	group := createTestGroup()

	procs := group.GetAllProcess("group1")

	if len(procs) != 2 || !util.HasAllElements(util.StringArrayToInterfacArray(procs), []interface{}{"proc1_1", "proc1_2"}) {
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

	if len(procs) != 2 || !util.HasAllElements(util.StringArrayToInterfacArray(procs), []interface{}{"proc2_2", "proc2_3"}) {
		t.Fail()
	}

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
	if len(added) != 1 || added[0] != "group-3" {
		t.Error("Fail to get the Added groups")
	}
	if len(changed) != 1 || changed[0] != "group-1" {
		t.Error("Fail to get changed groups")
	}

	if len(removed) != 1 || removed[0] != "group-2" {
		t.Error("Fail to get removed groups")
	}

}
