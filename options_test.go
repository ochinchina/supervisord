package main

import (
	"testing"

	"github.com/jessevdk/go-flags"
)

func parseOptions(t *testing.T, args ...string) Options {
	t.Helper()
	var opts Options
	p := flags.NewParser(&opts, flags.Default & ^flags.PrintErrors)
	p.Command.SubcommandsOptional = true
	if _, err := p.ParseArgs(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return opts
}

// Python supervisord's -n/--nodaemon and -e/--loglevel must be accepted so
// that command lines written for it keep working.
func TestPythonCompatFlags(t *testing.T) {
	tests := []struct {
		args      []string
		noDaemon  bool
		logLevel  string
		daemonize bool
	}{
		{args: []string{"-n", "-c", "/etc/supervisor/supervisord.conf"}, noDaemon: true},
		{args: []string{"--nodaemon"}, noDaemon: true},
		{args: []string{"-e", "warn"}, logLevel: "warn"},
		{args: []string{"--loglevel=debug"}, logLevel: "debug"},
		{args: []string{"-n", "-e", "info", "-c", "x.conf"}, noDaemon: true, logLevel: "info"},
		{args: []string{"-d"}, daemonize: true},
		{args: []string{"-d", "-n"}, noDaemon: true},
	}
	for _, tt := range tests {
		opts := parseOptions(t, tt.args...)
		if opts.NoDaemon != tt.noDaemon {
			t.Errorf("%v: NoDaemon = %v, want %v", tt.args, opts.NoDaemon, tt.noDaemon)
		}
		if opts.LogLevel != tt.logLevel {
			t.Errorf("%v: LogLevel = %q, want %q", tt.args, opts.LogLevel, tt.logLevel)
		}
		if got := opts.shouldDaemonize(); got != tt.daemonize {
			t.Errorf("%v: shouldDaemonize() = %v, want %v", tt.args, got, tt.daemonize)
		}
	}
}
