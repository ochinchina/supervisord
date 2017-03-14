package main

import (
	"testing"
)

// return true if the elem is in the array arr
func inArray(elem interface{}, arr []interface{}) bool {
	for _, e := range arr {
		if e == elem {
			return true
		}
	}
	return false
}

//return true if the array arr1 contains all elements of array arr2
func hasAllElements(arr1 []interface{}, arr2 []interface{}) bool {
	for _, e2 := range arr2 {
		if !inArray(e2, arr1) {
			return false
		}
	}
	return true
}

func stringArrayToInterfacArray(arr []string) []interface{} {
	result := make([]interface{}, 0)
	for _, s := range arr {
		result = append(result, s)
	}
	return result
}

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
