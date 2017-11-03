// +build windows,nacl,plan9

package main

func NewSysLogger(name string, logEventEmitter LogEventEmitter) *SysLogger {
	return &SysLogger{logEventEmitter: logEventEmitter, logWriter: nil}
}
