package logger

import (
	"fmt"

	"github.com/stuartcarnie/gopm/faults"
)

// NullLogger discard the program stdout/stderr log
type NullLogger struct{}

// NewNullLogger creates a NullLogger
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// SetPid set the pid of program
func (l *NullLogger) SetPid(pid int) {
}

// Write write the log to this logger
func (l *NullLogger) Write(p []byte) (int, error) {
	return len(p), nil
}

// Close close the logger
func (l *NullLogger) Close() error {
	return nil
}

// ReadLog read the log, return error
func (l *NullLogger) ReadLog(offset, length int64) (string, error) {
	return "", faults.NewFault(faults.NoFile, "NO_FILE")
}

// ReadTailLog tail the log, return error
func (l *NullLogger) ReadTailLog(offset, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NoFile, "NO_FILE")
}

// ClearCurLogFile close current log file, return error
func (l *NullLogger) ClearCurLogFile() error {
	return fmt.Errorf("no log")
}

// ClearAllLogFile clear all the lof file, return error
func (l *NullLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NoFile, "NO_FILE")
}
