package logger

import (
	"fmt"
	"sync"

	"github.com/stuartcarnie/gopm/faults"
)

// ChanLogger write log message by channel
type ChanLogger struct {
	recv chan []byte

	mu      sync.Mutex
	buffers [][]byte
}

// NewChanLogger create a ChanLogger object
func NewChanLogger(recv chan []byte) *ChanLogger {
	return &ChanLogger{recv: recv}
}

// SetPid set the program pid
func (l *ChanLogger) SetPid(pid int) {
}

// Write write the log to channel
func (l *ChanLogger) Write(p []byte) (int, error) {
	buf := l.getBuffer()
	buf = append(buf[:0], p...)

	l.recv <- buf
	return len(buf), nil
}

// Close close the channel
func (l *ChanLogger) Close() error {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	close(l.recv)
	return nil
}

// ReadLog read log, return error
func (l *ChanLogger) ReadLog(offset, length int64) (string, error) {
	return "", faults.NewFault(faults.NoFile, "NO_FILE")
}

// ReadTailLog tail the log, return error
func (l *ChanLogger) ReadTailLog(offset, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NoFile, "NO_FILE")
}

// ClearCurLogFile clear the log, return error
func (l *ChanLogger) ClearCurLogFile() error {
	return fmt.Errorf("no log")
}

// ClearAllLogFile clear the log, return error
func (l *ChanLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NoFile, "NO_FILE")
}

func (l *ChanLogger) getBuffer() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buffers) > 0 {
		end := len(l.buffers) - 1
		buf := l.buffers[end]
		l.buffers[end] = nil
		l.buffers = l.buffers[:end]
		return buf
	}
	return make([]byte, 0, 4096)
}

func (l *ChanLogger) PutBuffer(buf []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	buf = buf[:0]
	l.buffers = append(l.buffers, buf)
}
