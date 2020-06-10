package model

import "github.com/creasty/defaults"

type Program struct {
	Group string `yaml:"-"`

	Name                     string            `yaml:"name"`
	Directory                string            `yaml:"directory"`
	Command                  string            `yaml:"command"`
	Environment              map[string]string `yaml:"environment"`
	User                     string            `yaml:"user"`
	ExitCodes                []int             `yaml:"exit_codes" default:"[0,2]"`
	Priority                 int               `yaml:"priority" default:"999"`
	RestartPause             Duration          `yaml:"restart_pause"`
	StartRetries             int               `yaml:"start_retries" default:"3"`
	StartSeconds             Duration          `yaml:"start_seconds" default:"1000000000"`
	Cron                     string            `yaml:"cron"`
	AutoStart                bool              `yaml:"auto_start" default:"true"`
	AutoRestart              *bool             `yaml:"auto_restart"`
	RestartDirectoryMonitor  string            `yaml:"restart_directory_monitor"`
	RestartFilePattern       string            `yaml:"restart_file_pattern" default:"*"`
	RestartWhenBinaryChanged bool              `yaml:"restart_when_binary_changed"`
	StopSignals              []string          `yaml:"stop_signals"`
	StopWaitSeconds          Duration          `yaml:"stop_wait_seconds" default:"10000000000"`
	StopAsGroup              bool              `yaml:"stop_as_group"`
	KillAsGroup              bool              `yaml:"kill_as_group"`
	StdoutLogFile            string            `yaml:"stdout_logfile" default:"/dev/null"`
	StdoutLogfileBackups     int               `yaml:"stdout_logfile_backups" default:"10"`
	StdoutLogFileMaxBytes    int               `yaml:"stdout_logfile_max_bytes" default:"52428800"`
	RedirectStderr           bool              `yaml:"redirect_stderr"`
	StderrLogFile            string            `yaml:"stderr_logfile" default:"/dev/null"`
	StderrLogfileBackups     int               `yaml:"stderr_logfile_backups" default:"10"`
	StderrLogFileMaxBytes    int               `yaml:"stderr_logfile_max_bytes" default:"52428800"`
	DependsOn                []string          `yaml:"depends_on"`
}

func (p *Program) UnmarshalYAML(f func(interface{}) error) error {
	_ = defaults.Set(p)
	type tmp Program // avoid recursive calls to UnmarshalYAML
	return f((*tmp)(p))
}
