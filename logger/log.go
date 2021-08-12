package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/faults"
)

// Logger the log interface to log program stdout/stderr logs to file
type Logger interface {
	io.WriteCloser
	SetPid(pid int)
	ReadLog(offset int64, length int64) (string, error)
	ReadTailLog(offset int64, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

// LogEventEmitter the interface to emit log events
type LogEventEmitter interface {
	emitLogEvent(data string)
}

// FileLogger log program stdout/stderr to file
type FileLogger struct {
	name            string
	maxSize         int64
	backups         int
	fileSize        int64
	file            *os.File
	logEventEmitter LogEventEmitter
	locker          sync.Locker
}

// SysLogger log program stdout/stderr to syslog
type SysLogger struct {
	NullLogger
	logWriter       io.WriteCloser
	logEventEmitter LogEventEmitter
}

// NullLogger discard the program stdout/stderr log
type NullLogger struct {
	logEventEmitter LogEventEmitter
}

// NullLocker no lock
type NullLocker struct {
}

// ChanLogger write log message by channel
type ChanLogger struct {
	channel chan []byte
}

// CompositeLogger dispatch the log message to other loggers
type CompositeLogger struct {
	lock    sync.Mutex
	loggers []Logger
}

// NewFileLogger creates FileLogger object
func NewFileLogger(name string, maxSize int64, backups int, logEventEmitter LogEventEmitter, locker sync.Locker) *FileLogger {
	logger := &FileLogger{name: name,
		maxSize:         maxSize,
		backups:         backups,
		fileSize:        0,
		file:            nil,
		logEventEmitter: logEventEmitter,
		locker:          locker}
	logger.openFile(false)
	return logger
}

// SetPid sets pid of the program
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
		l.file, err = os.OpenFile(l.name, os.O_RDWR|os.O_APPEND, 0666)
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

// ClearCurLogFile clears contents (re-open with truncate) of current log file
func (l *FileLogger) ClearCurLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	return l.openFile(true)
}

// ClearAllLogFile clears contents of all log files (re-open with truncate)
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

// ReadLog reads log from current logfile
func (l *FileLogger) ReadLog(offset int64, length int64) (string, error) {
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

// ReadTailLog tails current log file
func (l *FileLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	if offset < 0 {
		return "", offset, false, fmt.Errorf("offset should not be less than 0")
	}
	if length < 0 {
		return "", offset, false, fmt.Errorf("length should be not be less than 0")
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

// Write overrides function in io.Writer. Write log message to the file
func (l *FileLogger) Write(p []byte) (int, error) {
	l.locker.Lock()
	defer l.locker.Unlock()

	n, err := l.file.Write(p)

	if err != nil {
		return n, err
	}
	l.logEventEmitter.emitLogEvent(string(p))
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

// Close file logger
func (l *FileLogger) Close() error {
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// Write log to syslog
func (sl *SysLogger) Write(b []byte) (int, error) {
	sl.logEventEmitter.emitLogEvent(string(b))
	if sl.logWriter == nil {
		return 0, errors.New("not connect to syslog server")
	}
	return sl.logWriter.Write(b)
}

// Close (sys)logger
func (sl *SysLogger) Close() error {
	if sl.logWriter == nil {
		return errors.New("not connect to syslog server")
	}
	return sl.logWriter.Close()
}

// NewNullLogger creates NullLogger object
func NewNullLogger(logEventEmitter LogEventEmitter) *NullLogger {
	return &NullLogger{logEventEmitter: logEventEmitter}
}

// SetPid sets pid of program
func (l *NullLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// Write log to NullLogger
func (l *NullLogger) Write(p []byte) (int, error) {
	l.logEventEmitter.emitLogEvent(string(p))
	return len(p), nil
}

// Close the NullLogger
func (l *NullLogger) Close() error {
	return nil
}

// ReadLog returns error for NullLogger
func (l *NullLogger) ReadLog(offset int64, length int64) (string, error) {
	return "", faults.NewFault(faults.NoFile, "NO_FILE")
}

// ReadTailLog returns error for NullLogger
func (l *NullLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NoFile, "NO_FILE")
}

// ClearCurLogFile returns error for NullLogger
func (l *NullLogger) ClearCurLogFile() error {
	return fmt.Errorf("No log")
}

// ClearAllLogFile returns error for NullLogger
func (l *NullLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NoFile, "NO_FILE")
}

// NewChanLogger creates ChanLogger object
func NewChanLogger(channel chan []byte) *ChanLogger {
	return &ChanLogger{channel: channel}
}

// SetPid sets program pid
func (l *ChanLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// Write log to the channel
func (l *ChanLogger) Write(p []byte) (int, error) {
	l.channel <- p
	return len(p), nil
}

// Close ChanLogger
func (l *ChanLogger) Close() error {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	close(l.channel)
	return nil
}

// ReadLog returns error for ChanLogger
func (l *ChanLogger) ReadLog(offset int64, length int64) (string, error) {
	return "", faults.NewFault(faults.NoFile, "NO_FILE")
}

// ReadTailLog returns error for ChanLogger
func (l *ChanLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NoFile, "NO_FILE")
}

// ClearCurLogFile returns error for ChanLogger
func (l *ChanLogger) ClearCurLogFile() error {
	return fmt.Errorf("No log")
}

// ClearAllLogFile returns error for ChanLogger
func (l *ChanLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NoFile, "NO_FILE")
}

// NewNullLocker creates new NullLocker object
func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

// Lock is a stub function for NullLocker
func (l *NullLocker) Lock() {
}

// Unlock is a stub function for NullLocker
func (l *NullLocker) Unlock() {
}

// StdLogger stdout/stderr logger implementation
type StdLogger struct {
	NullLogger
	logEventEmitter LogEventEmitter
	writer          io.Writer
}

// NewStdoutLogger creates StdLogger object
func NewStdoutLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{logEventEmitter: logEventEmitter,
		writer: os.Stdout}
}

// Write output to stdout/stderr
func (l *StdLogger) Write(p []byte) (int, error) {
	n, err := l.writer.Write(p)
	if err != nil {
		l.logEventEmitter.emitLogEvent(string(p))
	}
	return n, err
}

// NewStderrLogger creates stderr logger
func NewStderrLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{logEventEmitter: logEventEmitter,
		writer: os.Stderr}
}

// LogCaptureLogger capture the log for further analysis
type LogCaptureLogger struct {
	underlineLogger        Logger
	procCommEventCapWriter io.Writer
	procCommEventCapture   *events.ProcCommEventCapture
}

// NewLogCaptureLogger creates new LogCaptureLogger object
func NewLogCaptureLogger(underlineLogger Logger,
	captureMaxBytes int,
	stdType string,
	procName string,
	groupName string) *LogCaptureLogger {
	r, w := io.Pipe()
	eventCapture := events.NewProcCommEventCapture(r,
		captureMaxBytes,
		stdType,
		procName,
		groupName)
	return &LogCaptureLogger{underlineLogger: underlineLogger,
		procCommEventCapWriter: w,
		procCommEventCapture:   eventCapture}
}

// SetPid sets pid of program
func (l *LogCaptureLogger) SetPid(pid int) {
	l.procCommEventCapture.SetPid(pid)
}

// Write log to LogCaptureLogger
func (l *LogCaptureLogger) Write(p []byte) (int, error) {
	l.procCommEventCapWriter.Write(p)
	return l.underlineLogger.Write(p)
}

// Close LogCaptureLogger
func (l *LogCaptureLogger) Close() error {
	return l.underlineLogger.Close()
}

// ReadLog reads log from LogCaptureLogger
func (l *LogCaptureLogger) ReadLog(offset int64, length int64) (string, error) {
	return l.underlineLogger.ReadLog(offset, length)
}

// ReadTailLog tails log from LogCaptureLogger
func (l *LogCaptureLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return l.underlineLogger.ReadTailLog(offset, length)
}

// ClearCurLogFile clears current log file
func (l *LogCaptureLogger) ClearCurLogFile() error {
	return l.underlineLogger.ClearCurLogFile()
}

// ClearAllLogFile clears all log files
func (l *LogCaptureLogger) ClearAllLogFile() error {
	return l.underlineLogger.ClearAllLogFile()
}

// NullLogEventEmitter will not emit log to any listener
type NullLogEventEmitter struct {
}

// NewNullLogEventEmitter creates new NullLogEventEmitter object
func NewNullLogEventEmitter() *NullLogEventEmitter {
	return &NullLogEventEmitter{}
}

// emitLogEvent emit the log
func (ne *NullLogEventEmitter) emitLogEvent(data string) {
}

// StdLogEventEmitter emit the Stdout/Stderr LogEvent
type StdLogEventEmitter struct {
	Type        string
	processName string
	groupName   string
	pidFunc     func() int
}

// NewStdoutLogEventEmitter creates new StdLogEventEmitter object
func NewStdoutLogEventEmitter(processName string, groupName string, procPidFunc func() int) *StdLogEventEmitter {
	return &StdLogEventEmitter{Type: "stdout",
		processName: processName,
		groupName:   groupName,
		pidFunc:     procPidFunc}
}

// NewStderrLogEventEmitter creates new StdLogEventEmitter object for emitting Stderr log events
func NewStderrLogEventEmitter(processName string, groupName string, procPidFunc func() int) *StdLogEventEmitter {
	return &StdLogEventEmitter{Type: "stderr",
		processName: processName,
		groupName:   groupName,
		pidFunc:     procPidFunc}
}

// emitLogEvent emits stdout/stderr log event (with data)
func (se *StdLogEventEmitter) emitLogEvent(data string) {
	if se.Type == "stdout" {
		events.EmitEvent(events.CreateProcessLogStdoutEvent(se.processName, se.groupName, se.pidFunc(), data))
	} else {
		events.EmitEvent(events.CreateProcessLogStderrEvent(se.processName, se.groupName, se.pidFunc(), data))
	}
}

// BackgroundWriteCloser write data in background
type BackgroundWriteCloser struct {
	io.WriteCloser
	logChannel  chan []byte
	writeCloser io.WriteCloser
}

// NewBackgroundWriteCloser creates new BackgroundWriteCloser object
func NewBackgroundWriteCloser(writeCloser io.WriteCloser) *BackgroundWriteCloser {
	channel := make(chan []byte)
	bw := &BackgroundWriteCloser{logChannel: channel,
		writeCloser: writeCloser}

	bw.start()
	return bw
}

func (bw *BackgroundWriteCloser) start() {
	go func() {
		for {
			b, ok := <-bw.logChannel
			if !ok {
				break
			}
			bw.writeCloser.Write(b)
		}
	}()
}

// Write data in background
func (bw *BackgroundWriteCloser) Write(p []byte) (n int, err error) {
	bw.logChannel <- p
	return len(p), nil
}

// Close background data channel
func (bw *BackgroundWriteCloser) Close() error {
	close(bw.logChannel)
	return bw.writeCloser.Close()
}

// NewCompositeLogger creates new CompositeLogger object (pool of loggers)
func NewCompositeLogger(loggers []Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

// AddLogger adds logger to CompositeLogger pool
func (cl *CompositeLogger) AddLogger(logger Logger) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl.loggers = append(cl.loggers, logger)
}

// RemoveLogger removes logger from CompositeLogger pool
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

// Write dispatches log data to the loggers in CompositeLogger pool
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

// Close all loggers in CompositeLogger pool
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

// SetPid sets pid to all loggers in CompositeLogger pool
func (cl *CompositeLogger) SetPid(pid int) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for _, logger := range cl.loggers {
		logger.SetPid(pid)
	}
}

// ReadLog read log data from first logger in CompositeLogger pool
func (cl *CompositeLogger) ReadLog(offset int64, length int64) (string, error) {
	return cl.loggers[0].ReadLog(offset, length)
}

// ReadTailLog tail the log data from first logger in CompositeLogger pool
func (cl *CompositeLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return cl.loggers[0].ReadTailLog(offset, length)
}

// ClearCurLogFile clear the first logger file in CompositeLogger pool
func (cl *CompositeLogger) ClearCurLogFile() error {
	return cl.loggers[0].ClearCurLogFile()
}

// ClearAllLogFile clear all the files of first logger in CompositeLogger pool
func (cl *CompositeLogger) ClearAllLogFile() error {
	return cl.loggers[0].ClearAllLogFile()
}

// NewLogger creates logger for a program with parameters
func NewLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, props map[string]string, logEventEmitter LogEventEmitter) Logger {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for i, f := range files {
		var lr Logger
		if i == 0 {
			lr = createLogger(programName, f, locker, maxBytes, backups, props, logEventEmitter)
		} else {
			lr = createLogger(programName, f, NewNullLocker(), maxBytes, backups, props, NewNullLogEventEmitter())
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

func createLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, props map[string]string, logEventEmitter LogEventEmitter) Logger {
	if logFile == "/dev/stdout" {
		return NewStdoutLogger(logEventEmitter)
	}
	if logFile == "/dev/stderr" {
		return NewStderrLogger(logEventEmitter)
	}
	if logFile == "/dev/null" {
		return NewNullLogger(logEventEmitter)
	}

	if logFile == "syslog" {
		return NewSysLogger(programName, props, logEventEmitter)
	}
	if strings.HasPrefix(logFile, "syslog") {
		fields := strings.Split(logFile, "@")
		fields[0] = strings.TrimSpace(fields[0])
		fields[1] = strings.TrimSpace(fields[1])
		if len(fields) == 2 && fields[0] == "syslog" {
			return NewRemoteSysLogger(programName, fields[1], props, logEventEmitter)
		}
	}
	if len(logFile) > 0 {
		return NewFileLogger(logFile, maxBytes, backups, logEventEmitter, locker)
	}
	return NewNullLogger(logEventEmitter)
}
