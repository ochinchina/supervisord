package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ochinchina/supervisord/config"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// newTestProcess loads a single [program:name] section and returns its
// process. The caller stops it.
func newTestProcess(t *testing.T, name, section string) *Process {
	t.Helper()
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "supervisord.conf")
	if err := os.WriteFile(cfgFile, []byte(section), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.NewConfig(cfgFile)
	if _, err := cfg.Load(); err != nil {
		t.Fatalf("load config: %v", err)
	}
	for _, entry := range cfg.GetPrograms() {
		if entry.GetProgramName() == name {
			return NewProcess("supervisord", entry)
		}
	}
	t.Fatalf("program %s not found in config", name)
	return nil
}

// fatalCount returns how often the program went FATAL. The state is only
// FATAL for an instant before autorestart starts it again, so count the
// error failToStartProgram logs on every transition to FATAL.
func fatalCount(hook *logtest.Hook, name string) int {
	n := 0
	for _, e := range hook.AllEntries() {
		if e.Level == log.ErrorLevel && e.Data["program"] == name && strings.HasPrefix(e.Message, "fail to start program") {
			n++
		}
	}
	return n
}

func countLines(t *testing.T, file string) int {
	t.Helper()
	b, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(b), "\n")
}

// A program that exits after it reached RUNNING (it stayed up longer than
// startsecs) is restarted without counting towards startretries, as in
// Python supervisor. Only start failures within startsecs count, so it must
// never become FATAL however often it exits.
func TestExitAfterRunningDoesNotCountAsStartRetry(t *testing.T) {
	runs := filepath.Join(t.TempDir(), "runs")
	p := newTestProcess(t, "crasher", fmt.Sprintf(`[program:crasher]
command=/bin/sh -c "echo run >> %s; sleep 2.5; exit 1"
startsecs=1
startretries=2
autorestart=unexpected
`, runs))
	hook := logtest.NewGlobal()
	defer hook.Reset()
	p.Start(false)
	defer p.Stop(true)

	deadline := time.Now().Add(20 * time.Second)
	for countLines(t, runs) < 4 {
		if time.Now().After(deadline) {
			t.Fatalf("only %d runs within 20s, state %v", countLines(t, runs), p.GetState())
		}
		time.Sleep(100 * time.Millisecond)
	}
	if n := fatalCount(hook, "crasher"); n > 0 {
		t.Fatalf("program went FATAL %d times in %d runs; exits after RUNNING must not count as start retries", n, countLines(t, runs))
	}
}

// A program that exits before startsecs has failed to start; after
// startretries attempts it becomes FATAL.
func TestStartFailuresBecomeFatal(t *testing.T) {
	p := newTestProcess(t, "failer", `[program:failer]
command=/bin/sh -c "exit 1"
startsecs=1
startretries=2
autorestart=unexpected
`)
	hook := logtest.NewGlobal()
	defer hook.Reset()
	p.Start(false)
	defer p.Stop(true)

	deadline := time.Now().Add(10 * time.Second)
	for fatalCount(hook, "failer") == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("program did not become FATAL within 10s, state %v", p.GetState())
		}
		time.Sleep(50 * time.Millisecond)
	}
}
