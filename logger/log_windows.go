// +build windows plan9 nacl

package logger

func NewSysLogger(name string) *SysLogger {
	return &SysLogger{logWriter: nil}
}

func NewRemoteSysLogger(name, config string) *SysLogger {
	return NewSysLogger(name)
}
