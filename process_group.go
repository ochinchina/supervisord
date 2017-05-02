package main

type ProcessGroup struct {
	//mapping between the program and its group
	processGroup map[string]string
}

func NewProcessGroup() *ProcessGroup {
	return &ProcessGroup{make(map[string]string)}
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
	for group, _ := range groups {
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
	return defGroup
}
