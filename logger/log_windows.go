// +build windows plan9 nacl

package logger

func NewSysLogger(name string, logEventEmitter LogEventEmitter) *SysLogger {
	return &SysLogger{logEventEmitter: logEventEmitter, logWriter: nil}
}
