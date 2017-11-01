// +build !windows

package main

import (
	"errors"
	"log/syslog"
)

type SysLogger struct {
	NullLogger
	logWriter       *syslog.Writer
	logEventEmitter LogEventEmitter
}

func NewSysLogger(name string, logEventEmitter LogEventEmitter) *SysLogger {
	writer, err := syslog.New(syslog.LOG_DEBUG, name)
	logger := &SysLogger{logEventEmitter: logEventEmitter}
	if err == nil {
		logger.logWriter = writer
	}
	return logger
}

func (sl *SysLogger) Write(b []byte) (int, error) {
	sl.logEventEmitter.emitLogEvent(string(b))
	if sl.logWriter != nil {
		return sl.logWriter.Write(b)
	} else {
		return 0, errors.New("not connect to syslog server")
	}
}

func (sl *SysLogger) Close() error {
	if sl.logWriter != nil {
		return sl.logWriter.Close()
	} else {
		return errors.New("not connect to syslog server")
	}
}
