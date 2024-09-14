package types

import (
	"fmt"
)

// ProcessInfo the running process information
type ProcessInfo struct {
	Name          string `xml:"name" json:"name"`
	Group         string `xml:"group" json:"group"`
	Description   string `xml:"description" json:"description"`
	Start         int    `xml:"start" json:"start"`
	Stop          int    `xml:"stop" json:"stop"`
	Now           int    `xml:"now" json:"now"`
	State         int    `xml:"state" json:"state"`
	Statename     string `xml:"statename" json:"statename"`
	Spawnerr      string `xml:"spawnerr" json:"spawnerr"`
	Exitstatus    int    `xml:"exitstatus" json:"exitstatus"`
	Logfile       string `xml:"logfile" json:"logfile"`
	StdoutLogfile string `xml:"stdout_logfile" json:"stdout_logfile"`
	StderrLogfile string `xml:"stderr_logfile" json:"stderr_logfile"`
	Pid           int    `xml:"pid" json:"pid"`
}

// ReloadConfigResult the result of supervisor configuration reloading
type ReloadConfigResult struct {
	AddedGroup   []string
	ChangedGroup []string
	RemovedGroup []string
}

// ProcessSignal process signal includes program name and signal sent to it
type ProcessSignal struct {
	Name   string
	Signal string
}

// BooleanReply any rpc result with BooleanReply type
type BooleanReply struct {
	Success bool
}

// GetFullName returns full name of program including group and name
func (pi ProcessInfo) GetFullName() string {
	if len(pi.Group) > 0 {
		return fmt.Sprintf("%s:%s", pi.Group, pi.Name)
	}
	return pi.Name
}
