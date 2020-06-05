package logger

import "sync"

// CompositeLogger dispatch the log message to other loggers
type CompositeLogger struct {
	lock    sync.Mutex
	loggers []Logger
}

// NewCompositeLogger create a new CompositeLogger object
func NewCompositeLogger(loggers []Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

// AddLogger add a logger to receive the log data
func (cl *CompositeLogger) AddLogger(logger Logger) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl.loggers = append(cl.loggers, logger)
}

// RemoveLogger remove the previous added logger
func (cl *CompositeLogger) RemoveLogger(logger Logger) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	for i, t := range cl.loggers {
		if t == logger {
			cl.loggers = append(cl.loggers[:i], cl.loggers[i+1:]...)
			break
		}
	}
}

// Write dispatch the log data to loggers added by AddLogger() call
func (cl *CompositeLogger) Write(p []byte) (n int, err error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for i, logger := range cl.loggers {
		if i == 0 {
			n, err = logger.Write(p)
		} else {
			logger.Write(p)
		}
	}
	return
}

// Close close all the loggers added by AddLogger() call
func (cl *CompositeLogger) Close() (err error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for i, logger := range cl.loggers {
		if i == 0 {
			err = logger.Close()
		} else {
			logger.Close()
		}
	}
	return
}

// SetPid set pid to all the loggers added by AddLogger() call
func (cl *CompositeLogger) SetPid(pid int) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for _, logger := range cl.loggers {
		logger.SetPid(pid)
	}
}

// ReadLog read log data from first logger
func (cl *CompositeLogger) ReadLog(offset, length int64) (string, error) {
	return cl.loggers[0].ReadLog(offset, length)
}

// ReadTailLog tail the log data from first logger
func (cl *CompositeLogger) ReadTailLog(offset, length int64) (string, int64, bool, error) {
	return cl.loggers[0].ReadTailLog(offset, length)
}

// ClearCurLogFile clear the first logger file
func (cl *CompositeLogger) ClearCurLogFile() error {
	return cl.loggers[0].ClearCurLogFile()
}

// ClearAllLogFile clear all the files of first logger
func (cl *CompositeLogger) ClearAllLogFile() error {
	return cl.loggers[0].ClearAllLogFile()
}
