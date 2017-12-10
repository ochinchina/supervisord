// +build !windows,!nacl,!plan9

package logger

import (
	"log/syslog"
)

func NewSysLogger(name string, logEventEmitter LogEventEmitter) *SysLogger {
	writer, err := syslog.New(syslog.LOG_DEBUG, name)
	logger := &SysLogger{logEventEmitter: logEventEmitter}
	if err == nil {
		logger.logWriter = writer
	}
	return logger
}
