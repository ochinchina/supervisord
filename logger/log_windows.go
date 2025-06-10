//go:build windows || plan9 || nacl
// +build windows plan9 nacl

package logger

func NewSysLogger(name string, props map[string]string, logEventEmitter LogEventEmitter) *SysLogger {
	return &SysLogger{logEventEmitter: logEventEmitter, logWriter: nil}
}

func NewRemoteSysLogger(name string, config string, props map[string]string, logEventEmitter LogEventEmitter) *SysLogger {
	return NewSysLogger(name, props, logEventEmitter)
}
