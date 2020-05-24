package types

import (
	"fmt"
	"sort"
)

// ProcessInfo the running process information
type ProcessInfo struct {
	Name          string `json:"name"`
	Group         string `json:"group"`
	Description   string `json:"description"`
	Start         int64  `json:"start"`
	Stop          int64  `json:"stop"`
	Now           int64  `json:"now"`
	State         int64  `json:"state"`
	StateName     string `json:"statename"`
	SpawnErr      string `json:"spawnerr"`
	ExitStatus    int64  `json:"exitstatus"`
	Logfile       string `json:"logfile"`
	StdoutLogfile string `json:"stdout_logfile"`
	StderrLogfile string `json:"stderr_logfile"`
	Pid           int64  `json:"pid"`
}

// GetFullName get the full name of program includes group and name
func (pi ProcessInfo) GetFullName() string {
	if len(pi.Group) > 0 {
		return fmt.Sprintf("%s:%s", pi.Group, pi.Name)
	}
	return pi.Name
}

type ProcessInfos []ProcessInfo

func (pi ProcessInfos) SortByName() {
	sort.Slice(pi, func(i, j int) bool {
		return pi[i].Name < pi[j].Name
	})
}
