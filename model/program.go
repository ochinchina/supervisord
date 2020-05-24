package model

import "github.com/creasty/defaults"

type Program struct {
	Group string `yaml:"-" ini:"-"`

	Name                     string   `yaml:"name" ini:"-"`
	Directory                string   `yaml:"directory" ini:"directory"`
	Command                  string   `yaml:"command" ini:"command"`
	Environment              []string `yaml:"environment" ini:"environment" delim:"\n"`
	User                     string   `yaml:"user" ini:"user"`
	ExitCodes                []int    `yaml:"exit_codes" ini:"exitcodes" delim:"," default:"[0,2]"`
	Priority                 int      `yaml:"priority" ini:"priority" default:"999"`
	RestartPause             Duration `yaml:"restart_pause" ini:"restartpause"`
	StartRetries             int      `yaml:"start_retries" ini:"startretries" default:"3"`
	StartSeconds             Duration `yaml:"start_seconds" ini:"startsecs" default:"1000000000"`
	Cron                     string   `yaml:"cron" ini:"cron"`
	AutoStart                bool     `yaml:"auto_start" ini:"autostart" default:"true"`
	AutoRestart              *bool    `yaml:"auto_restart" ini:"autorestart"`
	RestartDirectoryMonitor  string   `yaml:"restart_directory_monitor" ini:"restart_directory_monitor"`
	RestartFilePattern       string   `yaml:"restart_file_pattern" ini:"restart_filePattern" default:"*"`
	RestartWhenBinaryChanged bool     `yaml:"restart_when_binary_changed" ini:"restart_when_binary_changed"`
	StopSignals              []string `yaml:"stop_signals" ini:"stopsignal" delim:" "`
	StopWaitSeconds          Duration `yaml:"stop_wait_seconds" ini:"stopwaitsecs" default:"10000000000"`
	StopAsGroup              bool     `yaml:"stop_as_group" ini:"stopasgroup"`
	KillAsGroup              bool     `yaml:"kill_as_group" ini:"killasgroup"`
	StdoutLogFile            string   `yaml:"stdout_logfile" ini:"stdout_logfile" default:"/dev/null"`
	StdoutLogfileBackups     int      `yaml:"stdout_logfile_backups" ini:"stdout_logfile_backups" default:"10"`
	StdoutLogFileMaxBytes    int      `yaml:"stdout_logfile_max_bytes" ini:"stdout_logfile_maxbytes" default:"52428800"`
	RedirectStderr           bool     `yaml:"redirect_stderr" ini:"redirect_stderr"`
	StderrLogFile            string   `yaml:"stderr_logfile" ini:"stderr_logfile" default:"/dev/null"`
	StderrLogfileBackups     int      `yaml:"stderr_logfile_backups" ini:"stderr_logfile_backups" default:"10"`
	StderrLogFileMaxBytes    int      `yaml:"stderr_logfile_max_bytes" ini:"stderr_logfile_maxbytes" default:"52428800"`
	DependsOn                []string `yaml:"depends_on" ini:"depends_on" delim:","`
}

func (p *Program) UnmarshalYAML(f func(interface{}) error) error {
	_ = defaults.Set(p)
	type tmp Program // avoid recursive calls to UnmarshalYAML
	return f((*tmp)(p))
}

func (p *Program) IsProgram() bool {
	return true
}

type Programs []*Program

func (p Programs) Sorted() Programs {
	var res ProgramByPriority
	for _, p := range p {
		if p.IsProgram() {
			res = append(res, p)
		}
	}
	return NewProcessSorter().SortProgram(res)
}

func (p Programs) Names() []string {
	programs := p.Sorted()
	names := make([]string, len(p))
	for i, program := range programs {
		names[i] = program.Name
	}
	return names
}

func (p Programs) GetProgramNames() []string {
	return p.Names()
}
