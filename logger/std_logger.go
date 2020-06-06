package logger

import (
	"io"
	"os"
)

// StdLogger stdout/stderr logger implementation
type StdLogger struct {
	NullLogger
	writer io.Writer
}

// NewStdoutLogger create a StdLogger object
func NewStdoutLogger() *StdLogger {
	return &StdLogger{writer: os.Stdout}
}

// Write output the log to stdout/stderr
func (l *StdLogger) Write(p []byte) (int, error) {
	return l.writer.Write(p)
}

// NewStderrLogger create a stderr logger
func NewStderrLogger() *StdLogger {
	return &StdLogger{writer: os.Stderr}
}
