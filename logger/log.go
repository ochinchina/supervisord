package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/stuartcarnie/gopm/faults"
)

// Logger the log interface to log program stdout/stderr logs to file
type Logger interface {
	io.WriteCloser
	SetPid(pid int)
	ReadLog(offset, length int64) (string, error)
	ReadTailLog(offset, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

// FileLogger log program stdout/stderr to file
type FileLogger struct {
	name     string
	maxSize  int64
	backups  int
	fileSize int64
	file     *os.File
	locker   sync.Locker
}

// NullLogger discard the program stdout/stderr log
type NullLogger struct{}

// NullLocker no lock
type NullLocker struct{}

// ChanLogger write log message by channel
type ChanLogger struct {
	channel chan []byte
}

// CompositeLogger dispatch the log message to other loggers
type CompositeLogger struct {
	lock    sync.Mutex
	loggers []Logger
}

// NewFileLogger create a FileLogger object
func NewFileLogger(name string, maxSize int64, backups int, locker sync.Locker) *FileLogger {
	logger := &FileLogger{
		name:     name,
		maxSize:  maxSize,
		backups:  backups,
		fileSize: 0,
		file:     nil,
		locker:   locker,
	}
	logger.openFile(false)
	return logger
}

// SetPid set the pid of the program
func (l *FileLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// open the file and truncate the file if trunc is true
func (l *FileLogger) openFile(trunc bool) error {
	if l.file != nil {
		l.file.Close()
	}
	var err error
	fileInfo, err := os.Stat(l.name)

	if trunc || err != nil {
		l.file, err = os.Create(l.name)
	} else {
		l.fileSize = fileInfo.Size()
		l.file, err = os.OpenFile(l.name, os.O_RDWR|os.O_APPEND, 0o666)
	}
	if err != nil {
		fmt.Printf("Fail to open log file --%s-- with error %v\n", l.name, err)
	}
	return err
}

func (l *FileLogger) backupFiles() {
	for i := l.backups - 1; i > 0; i-- {
		src := fmt.Sprintf("%s.%d", l.name, i)
		dest := fmt.Sprintf("%s.%d", l.name, i+1)
		if _, err := os.Stat(src); err == nil {
			os.Rename(src, dest)
		}
	}
	dest := fmt.Sprintf("%s.1", l.name)
	os.Rename(l.name, dest)
}

// ClearCurLogFile clear the current log file contents
func (l *FileLogger) ClearCurLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	return l.openFile(true)
}

// ClearAllLogFile clear all the log files
func (l *FileLogger) ClearAllLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	for i := l.backups; i > 0; i-- {
		logFile := fmt.Sprintf("%s.%d", l.name, i)
		_, err := os.Stat(logFile)
		if err == nil {
			err = os.Remove(logFile)
			if err != nil {
				return faults.NewFault(faults.Failed, err.Error())
			}
		}
	}
	err := l.openFile(true)
	if err != nil {
		return faults.NewFault(faults.Failed, err.Error())
	}
	return nil
}

// ReadLog read the log from current logfile
func (l *FileLogger) ReadLog(offset, length int64) (string, error) {
	if offset < 0 && length != 0 {
		return "", faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
	}
	if offset >= 0 && length < 0 {
		return "", faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
	}

	l.locker.Lock()
	defer l.locker.Unlock()
	f, err := os.Open(l.name)
	if err != nil {
		return "", faults.NewFault(faults.Failed, "FAILED")
	}
	defer f.Close()

	// check the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", faults.NewFault(faults.Failed, "FAILED")
	}

	fileLen := statInfo.Size()

	if offset < 0 { // offset < 0 && length == 0
		offset = fileLen + offset
		if offset < 0 {
			offset = 0
		}
		length = fileLen - offset
	} else if length == 0 { // offset >= 0 && length == 0
		if offset > fileLen {
			return "", nil
		}
		length = fileLen - offset
	} else { // offset >= 0 && length > 0

		// if the offset exceeds the length of file
		if offset >= fileLen {
			return "", nil
		}

		// compute actual bytes should be read

		if offset+length > fileLen {
			length = fileLen - offset
		}
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", faults.NewFault(faults.Failed, "FAILED")
	}
	return string(b[:n]), nil
}

// ReadTailLog tail the log of current log file
func (l *FileLogger) ReadTailLog(offset, length int64) (string, int64, bool, error) {
	if offset < 0 {
		return "", offset, false, fmt.Errorf("invalid offset: value ≥ 0")
	}
	if length < 0 {
		return "", offset, false, fmt.Errorf("invalid length: value ≥ 0")
	}
	l.locker.Lock()
	defer l.locker.Unlock()

	// open the file
	f, err := os.Open(l.name)
	if err != nil {
		return "", 0, false, err
	}

	defer f.Close()

	// get the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}

	fileLen := statInfo.Size()

	// check if offset exceeds the length of file
	if offset >= fileLen {
		return "", fileLen, true, nil
	}

	// get the length
	if offset+length > fileLen {
		length = fileLen - offset
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", offset, false, err
	}
	return string(b[:n]), offset + int64(n), false, nil
}

// Write Override the function in io.Writer. Write the log message to the file
func (l *FileLogger) Write(p []byte) (int, error) {
	l.locker.Lock()
	defer l.locker.Unlock()

	n, err := l.file.Write(p)
	if err != nil {
		return n, err
	}
	l.fileSize += int64(n)
	if l.fileSize >= l.maxSize {
		fileInfo, errStat := os.Stat(l.name)
		if errStat == nil {
			l.fileSize = fileInfo.Size()
		} else {
			return n, errStat
		}
	}
	if l.fileSize >= l.maxSize {
		l.Close()
		l.backupFiles()
		l.openFile(true)
	}
	return n, err
}

// Close close the file logger
func (l *FileLogger) Close() error {
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// NewNullLogger creates a NullLogger
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// SetPid set the pid of program
func (l *NullLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// Write write the log to this logger
func (l *NullLogger) Write(p []byte) (int, error) {
	return len(p), nil
}

// Close close the logger
func (l *NullLogger) Close() error {
	return nil
}

// ReadLog read the log, return error
func (l *NullLogger) ReadLog(offset, length int64) (string, error) {
	return "", faults.NewFault(faults.NoFile, "NO_FILE")
}

// ReadTailLog tail the log, return error
func (l *NullLogger) ReadTailLog(offset, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NoFile, "NO_FILE")
}

// ClearCurLogFile close current log file, return error
func (l *NullLogger) ClearCurLogFile() error {
	return fmt.Errorf("no log")
}

// ClearAllLogFile clear all the lof file, return error
func (l *NullLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NoFile, "NO_FILE")
}

// NewChanLogger create a ChanLogger object
func NewChanLogger(channel chan []byte) *ChanLogger {
	return &ChanLogger{channel: channel}
}

// SetPid set the program pid
func (l *ChanLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// Write write the log to channel
func (l *ChanLogger) Write(p []byte) (int, error) {
	l.channel <- p
	return len(p), nil
}

// Close close the channel
func (l *ChanLogger) Close() error {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	close(l.channel)
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

// NewNullLocker create a new NullLocker object
func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

// Lock acquire the lock
func (l *NullLocker) Lock() {
}

// Unlock release the lock
func (l *NullLocker) Unlock() {
}

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

// NewLogger create a logger for a program with parameters
//
func NewLogger(programName, logFile string, locker sync.Locker, maxBytes int64, backups int) Logger {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for i, f := range files {
		var lr Logger
		if i == 0 {
			lr = createLogger(programName, f, locker, maxBytes, backups)
		} else {
			lr = createLogger(programName, f, NewNullLocker(), maxBytes, backups)
		}
		loggers = append(loggers, lr)
	}
	return NewCompositeLogger(loggers)
}

func splitLogFile(logFile string) []string {
	files := strings.Split(logFile, ",")
	for i, f := range files {
		files[i] = strings.TrimSpace(f)
	}
	return files
}

func createLogger(programName, logFile string, locker sync.Locker, maxBytes int64, backups int) Logger {
	if logFile == "/dev/stdout" {
		return NewStdoutLogger()
	}
	if logFile == "/dev/stderr" {
		return NewStderrLogger()
	}
	if logFile == "/dev/null" {
		return NewNullLogger()
	}

	if len(logFile) > 0 {
		return NewFileLogger(logFile, maxBytes, backups, locker)
	}
	return NewNullLogger()
}
