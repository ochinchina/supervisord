package config

import (
	"bytes"
	"strings"
	"github.com/ochinchina/supervisord/util"
)

type ProcessGroup struct {
	//mapping between the program and its group
	processGroup map[string]string
}

func NewProcessGroup() *ProcessGroup {
	return &ProcessGroup{processGroup: make(map[string]string)}
}

// clone the process group
func (pg *ProcessGroup) Clone() *ProcessGroup {
	new_pg := NewProcessGroup()
	for k, v := range pg.processGroup {
		new_pg.processGroup[k] = v
	}
	return new_pg
}

func (pg *ProcessGroup) Sub(other *ProcessGroup) (added []string, changed []string, removed []string) {
	thisGroup := pg.GetAllGroup()
	otherGroup := other.GetAllGroup()
	added = util.Sub(thisGroup, otherGroup)
	changed = make([]string, 0)
	removed = util.Sub(otherGroup, thisGroup)

	for _, group := range thisGroup {
		proc_1 := pg.GetAllProcess(group)
		proc_2 := other.GetAllProcess(group)
		if len(proc_2) > 0 && !util.IsSameStringArray(proc_1, proc_2) {
			changed = append(changed, group)
		}
	}
	return
}

//add a process to a group
func (pg *ProcessGroup) Add(group string, procName string) {
	pg.processGroup[procName] = group
}

//remove a process
func (pg *ProcessGroup) Remove(procName string) {
	delete(pg.processGroup, procName)
}

//get all the groups
func (pg *ProcessGroup) GetAllGroup() []string {
	groups := make(map[string]bool)
	for _, group := range pg.processGroup {
		groups[group] = true
	}

	result := make([]string, 0)
	for group := range groups {
		result = append(result, group)
	}
	return result
}

// get all the processes in a group
func (pg *ProcessGroup) GetAllProcess(group string) []string {
	result := make([]string, 0)
	for procName, groupName := range pg.processGroup {
		if group == groupName {
			result = append(result, procName)
		}
	}
	return result
}

// check if a process belongs to a group or not
func (pg *ProcessGroup) InGroup(procName string, group string) bool {
	groupName, ok := pg.processGroup[procName]
	if ok && group == groupName {
		return true
	}
	return false
}

func (pg *ProcessGroup) ForEachProcess(procFunc func(group string, procName string)) {
	for procName, groupName := range pg.processGroup {
		procFunc(groupName, procName)
	}
}

func (pg *ProcessGroup) GetGroup(procName string, defGroup string) string {
	group, ok := pg.processGroup[procName]

	if ok {
		return group
	}
	pg.processGroup[procName] = defGroup
	return defGroup
}

func (pg *ProcessGroup) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	for _, group := range pg.GetAllGroup() {
		buf.WriteString(group)
		buf.WriteString(":")
		buf.WriteString(strings.Join(pg.GetAllProcess(group), ","))
		buf.WriteString(";")
	}
	return buf.String()
}
