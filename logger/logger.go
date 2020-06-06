package logger

import (
	"io"
	"strings"
	"sync"
)

// Logger the log interface to log program stdout/stderr logs to file
type Logger interface {
	io.WriteCloser
	SetPid(pid int)
	ReadLog(offset, length int64) (string, error)
	ReadTailLog(offset, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

// NullLocker no lock
type NullLocker struct{}

// NewNullLocker create a new NullLocker object
func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

// Lock acquire the lock
func (l *NullLocker) Lock() {
}

// Unlock release the lock
func (l *NullLocker) Unlock() {
}

// NewLogger create a logger for a program with parameters
func NewLogger(programName, logFile string, locker sync.Locker, maxBytes int64, backups int) Logger {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for i, f := range files {
		var lr Logger
		if i == 0 {
			lr = createLogger(programName, f, locker, maxBytes, backups)
		} else {
			lr = createLogger(programName, f, NewNullLocker(), maxBytes, backups)
		}
		loggers = append(loggers, lr)
	}
	return NewCompositeLogger(loggers)
}

func splitLogFile(logFile string) []string {
	files := strings.Split(logFile, ",")
	for i, f := range files {
		files[i] = strings.TrimSpace(f)
	}
	return files
}

func createLogger(programName, logFile string, locker sync.Locker, maxBytes int64, backups int) Logger {
	if logFile == "/dev/stdout" {
		return NewStdoutLogger()
	}
	if logFile == "/dev/stderr" {
		return NewStderrLogger()
	}
	if logFile == "/dev/null" {
		return NewNullLogger()
	}

	if len(logFile) > 0 {
		return NewFileLogger(logFile, maxBytes, backups, locker)
	}
	return NewNullLogger()
}
