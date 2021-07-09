package config

import (
	"bytes"
	"strings"

	"github.com/ochinchina/supervisord/util"
)

// ProcessGroup manage the program and its group mapping
type ProcessGroup struct {
	// mapping between the program and its group
	processGroup map[string]string
}

// NewProcessGroup create a ProcessGroup object
func NewProcessGroup() *ProcessGroup {
	return &ProcessGroup{processGroup: make(map[string]string)}
}

// Clone clones process group
func (pg *ProcessGroup) Clone() *ProcessGroup {
	newPg := NewProcessGroup()
	for k, v := range pg.processGroup {
		newPg.processGroup[k] = v
	}
	return newPg
}

// Sub removes all the programs listed in other ProcessGroup from this ProcessGroup
func (pg *ProcessGroup) Sub(other *ProcessGroup) (added []string, changed []string, removed []string) {
	thisGroup := pg.GetAllGroup()
	otherGroup := other.GetAllGroup()
	added = util.Sub(thisGroup, otherGroup)
	changed = make([]string, 0)
	removed = util.Sub(otherGroup, thisGroup)

	for _, group := range thisGroup {
		proc1 := pg.GetAllProcess(group)
		proc2 := other.GetAllProcess(group)
		if len(proc2) > 0 && !util.IsSameStringArray(proc1, proc2) {
			changed = append(changed, group)
		}
	}
	return
}

// Add adds process to a group
func (pg *ProcessGroup) Add(group string, procName string) {
	pg.processGroup[procName] = group
}

// Remove removes a process from a group
func (pg *ProcessGroup) Remove(procName string) {
	delete(pg.processGroup, procName)
}

// GetAllGroup gets all the process groups
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

// GetAllProcess gets all the processes in a group
func (pg *ProcessGroup) GetAllProcess(group string) []string {
	result := make([]string, 0)
	for procName, groupName := range pg.processGroup {
		if group == groupName {
			result = append(result, procName)
		}
	}
	return result
}

// InGroup checks if process belongs to a group or not
func (pg *ProcessGroup) InGroup(procName string, group string) bool {
	groupName, ok := pg.processGroup[procName]
	if ok && group == groupName {
		return true
	}
	return false
}

// ForEachProcess iterates all the processes and process it with procFunc
func (pg *ProcessGroup) ForEachProcess(procFunc func(group string, procName string)) {
	for procName, groupName := range pg.processGroup {
		procFunc(groupName, procName)
	}
}

// GetGroup gets group name of process. If group was not found by
// procName, set its group to defGroup and return this defGroup
func (pg *ProcessGroup) GetGroup(procName string, defGroup string) string {
	group, ok := pg.processGroup[procName]

	if ok {
		return group
	}
	pg.processGroup[procName] = defGroup
	return defGroup
}

// String converts process and its group mapping to human-readable string
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
