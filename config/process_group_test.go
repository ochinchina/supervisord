package config

import (
	"github.com/ochinchina/supervisord/util"
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
	group_1 := NewProcessGroup()
	group_1.Add("group-1", "proc-11")
	group_1.Add("group-1", "proc-12")
	group_1.Add("group-2", "proc-21")

	group_2 := NewProcessGroup()
	group_2.Add("group-1", "proc-11")
	group_2.Add("group-1", "proc-12")
	group_2.Add("group-1", "proc-13")
	group_2.Add("group-3", "proc-31")

	added, changed, removed := group_2.Sub(group_1)
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
