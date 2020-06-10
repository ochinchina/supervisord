package config

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stuartcarnie/gopm/pkg/env"
)

type AutoStartMode int

const (
	AutoStartModeDefault AutoStartMode = iota
	AutoStartModeAlways
	AutoStartModeNever
)

type Process struct {
	Group                    string
	Name                     string
	Directory                string
	Command                  string
	Environment              map[string]string
	User                     string
	ExitCodes                []int
	Priority                 int
	RestartPause             time.Duration
	StartRetries             int
	StartSeconds             time.Duration
	Cron                     string
	AutoStart                bool
	AutoRestart              AutoStartMode
	RestartDirectoryMonitor  string
	RestartFilePattern       string
	RestartWhenBinaryChanged bool
	StopSignals              []string
	StopWaitSeconds          time.Duration
	StopAsGroup              bool
	KillAsGroup              bool
	StdoutLogFile            string
	StdoutLogfileBackups     int
	StdoutLogFileMaxBytes    int
	RedirectStderr           bool
	StderrLogFile            string
	StderrLogfileBackups     int
	StderrLogFileMaxBytes    int
	DependsOn                []string
}

var (
	cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
)

func (p *Process) CronSchedule() cron.Schedule {
	if len(p.Cron) == 0 {
		return nil
	}
	s, err := cronParser.Parse(p.Cron)
	if err != nil {
		panic(err)
	}
	return s
}

type Processes []*Process

func (p Processes) Sorted() Processes {
	res := make(ProcessByPriority, 0, len(p))
	for _, p := range p {
		res = append(res, p)
	}
	return NewProcessSorter().Sort(res)
}

func (p Processes) Names() []string {
	sorted := p.Sorted()
	names := make([]string, len(p))
	for i, proc := range sorted {
		names[i] = proc.Name
	}
	return names
}

func (p Processes) GetNames() []string {
	return p.Names()
}

type Group struct {
	Name     string
	Programs []string
}

type Environment struct {
	Path      string
	KeyValues env.KeyValues
}

type File struct {
	Root    string
	Name    string
	Path    string
	Content string
}

type LocalFile struct {
	Name     string
	FullPath string
	Hash     []byte
}

type Server struct {
	Name     string
	Address  string
	Username string
	Password string
}
