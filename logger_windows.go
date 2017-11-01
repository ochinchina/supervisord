// +build windows

package main

type SysLogger struct {
	NullLogger
	logEventEmitter LogEventEmitter
}

func NewSysLogger(name string, logEventEmitter LogEventEmitter) *SysLogger {
	return &SysLogger{
		logEventEmitter: logEventEmitter,
	}
}
