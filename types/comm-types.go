package types

type ProcessInfo struct {
	Name           string `xml:"name"`
	Group          string `xml:"group"`
	Description    string `xml:"description"`
	Start          int    `xml:"start"`
	Stop           int    `xml:"stop"`
	Now            int    `xml:"now"`
	State          int    `xml:"state"`
	Statename      string `xml:"statename"`
	Spawnerr       string `xml:"spawnerr"`
	Exitstatus     int    `xml:"exitstatus"`
	Logfile        string `xml:"logfile"`
	Stdout_logfile string `xml:"stdout_logfile"`
	Stderr_logfile string `xml:"stderr_logfile"`
	Pid            int    `xml:"pid"`
}

type ReloadConfigResult struct {
	AddedGroup   []string
	ChangedGroup []string
	RemovedGroup []string
}
