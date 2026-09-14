package logger

import (
	"container/ring"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/faults"
	log "github.com/sirupsen/logrus"
)

type ChainLogTail struct {
	lock         sync.Mutex
	logTailFuncs []func(log []byte)
}

func NewChainLogTail() *ChainLogTail {
	return &ChainLogTail{logTailFuncs: make([]func(log []byte), 0)}
}

func (clt *ChainLogTail) AddLogTail(logTailFunc func(log []byte)) error {
	clt.lock.Lock()
	defer clt.lock.Unlock()

	clt.logTailFuncs = append(clt.logTailFuncs, logTailFunc)
	return nil
}

func (clt *ChainLogTail) RemoveLogTail(logTailFunc func(log []byte)) error {
	clt.lock.Lock()
	defer clt.lock.Unlock()

	for i, f := range clt.logTailFuncs {
		if fmt.Sprintf("%p", f) == fmt.Sprintf("%p", logTailFunc) {
			clt.logTailFuncs = append(clt.logTailFuncs[:i], clt.logTailFuncs[i+1:]...)
			return nil
		}
	}
	return errors.New("log tail function not found")
}

func (clt *ChainLogTail) EmitLogTail(log []byte) {
	clt.lock.Lock()
	funcs := make([]func(log []byte), len(clt.logTailFuncs))
	copy(funcs, clt.logTailFuncs)
	clt.lock.Unlock()

	for _, f := range funcs {
		f(log)
	}
}

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
	name                  string
	currentLogFileName    string
	maxSize               int64
	backups               int
	fileSize              int64
	fileNameWithTimestamp bool
	file                  *os.File
	logEventEmitter       LogEventEmitter
	locker                sync.Locker
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
	once    sync.Once
}

// CompositeLogger dispatch the log message to other loggers
type CompositeLogger struct {
	lock    sync.Mutex
	loggers []Logger
}

type MemoryLogger struct {
	lock            sync.Mutex
	logEventEmitter LogEventEmitter
	logs            *ring.Ring
}

// NewFileLogger creates FileLogger object
func NewFileLogger(name string, maxSize int64, backups int, fileNameWithTimestamp bool, logEventEmitter LogEventEmitter, locker sync.Locker) *FileLogger {
	logger := &FileLogger{name: name,
		currentLogFileName:    "",
		maxSize:               maxSize,
		backups:               backups,
		fileSize:              0,
		file:                  nil,
		fileNameWithTimestamp: fileNameWithTimestamp,
		logEventEmitter:       logEventEmitter,
		locker:                locker}
	logger.backupFiles()
	logger.openFile(false)
	return logger
}

// SetPid sets pid of the program
func (l *FileLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// open the file and truncate the file if trunc is true
func (l *FileLogger) openFile(trunc bool) error {
	if l.backups <= 0 {
		return errors.New("backups is less than or equal to 0, no log file will be created")
	}
	if l.file != nil {
		l.file.Close()
	}
	if l.fileNameWithTimestamp {
		l.currentLogFileName = fmt.Sprintf("%s.%s", l.name, time.Now().Format("2006-01-02T15-04-05"))

		// if the size of latest one is less than maxSize, reuse it
		files := l.getTimestampedFiles()
		n := len(files)
		if n > 0 {
			fileInfo, err := os.Stat(files[n-1])
			if err == nil && fileInfo.Size() < l.maxSize {
				l.currentLogFileName = files[n-1]
			}
		}

		var err error
		fileInfo, err := os.Stat(l.currentLogFileName)
		if trunc || err != nil {
			l.file, err = os.Create(l.currentLogFileName)
			if err != nil {
				log.WithField("error", err).Errorf("Fail to create log file %s", l.currentLogFileName)
			}
		} else {
			l.fileSize = fileInfo.Size()
			l.file, err = os.OpenFile(l.currentLogFileName, os.O_RDWR|os.O_APPEND, 0666)
			if err != nil {
				log.WithField("error", err).Errorf("Fail to open log file %s", l.currentLogFileName)
			}
		}

		return err
	} else {
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
}

func (l *FileLogger) getTimestampedFiles() []string {
	absPath, err := filepath.Abs(l.name)
	if err != nil {
		absPath = l.name
	}

	entries, err := os.ReadDir(filepath.Dir(absPath))
	if err != nil {
		fmt.Printf("Fail to read log directory --%s-- with error %v\n", filepath.Dir(absPath), err)
		return make([]string, 0)
	} else {
		files := make([]string, 0)
		for _, entry := range entries {

			if l.isTimestampedFileName(entry.Name()) {
				files = append(files, filepath.Join(filepath.Dir(absPath), entry.Name()))
			}
		}

		sort.Strings(files)

		return files
	}

}
func (l *FileLogger) isTimestampedFileName(fileName string) bool {

	if !l.fileNameWithTimestamp {
		return false
	}
	baseName := filepath.Base(l.name)
	if !strings.HasPrefix(fileName, baseName) || fileName == baseName || len(fileName) <= len(baseName)+1 {
		return false
	}

	timestampPart := fileName[len(baseName)+1:]
	_, err := time.Parse("2006-01-02T15-04-05", timestampPart)

	return err == nil
}

func (l *FileLogger) backupFiles() {

	if l.fileNameWithTimestamp {
		files := l.getTimestampedFiles()
		if len(files) > l.backups {
			for _, f := range files[:len(files)-l.backups+1] {
				err := os.Remove(f)
				if err != nil {
					fmt.Printf("Fail to remove log file --%s-- with error %v\n", f, err)
				}
			}
		}

	} else {
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

	if l.fileNameWithTimestamp {
		absPath, err := filepath.Abs(l.name)
		if err != nil {
			absPath = l.name
		}
		entries, err := os.ReadDir(filepath.Dir(absPath))
		if err != nil {
			fmt.Printf("Fail to read log directory --%s-- with error %v\n", filepath.Dir(absPath), err)
		} else {
			files := make([]string, 0)
			for _, entry := range entries {
				if l.isTimestampedFileName(entry.Name()) {
					files = append(files, filepath.Join(filepath.Dir(absPath), entry.Name()))
				}
			}

			for _, f := range files {
				err := os.Remove(f)
				if err != nil {
					return faults.NewFault(faults.Failed, err.Error())
				}
			}
		}
	} else {

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
	}
	err := l.openFile(true)
	if err != nil {
		return faults.NewFault(faults.Failed, err.Error())
	}
	return nil
}

// ReadLog reads log from current logfile
func (l *FileLogger) ReadLog(offset int64, length int64) (string, error) {
	if l.backups <= 0 {
		return "", faults.NewFault(faults.NoFile, "NO_FILE")
	}

	l.locker.Lock()
	defer l.locker.Unlock()
	name := l.name
	if l.fileNameWithTimestamp {
		name = l.currentLogFileName
	}
	f, err := os.Open(name)

	if err != nil {
		log.WithField("error", err).Errorf("Fail to open log file %s", name)
		return "", faults.NewFault(faults.Failed, "FAILED")
	}
	defer f.Close()

	// check the length of file
	statInfo, err := f.Stat()
	if err != nil {
		log.WithField("error", err).Errorf("Fail to get file info for %s", name)
		return "", faults.NewFault(faults.Failed, "FAILED")
	}

	fileLen := statInfo.Size()

	if offset < 0 {
		if length != 0 {
			return "", faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
		}
		absOffset := offset * -1

		pos := max(fileLen-absOffset, 0)
		length = fileLen - pos
		b := make([]byte, length)
		n, err := f.ReadAt(b, pos)
		if err != nil {
			log.WithField("error", err).Errorf("Fail to read log file %s at offset %d with length %d", name, pos, length)
			return "", faults.NewFault(faults.Failed, "FAILED")
		}
		return string(b[:n]), nil

	} else {
		if length < 0 {
			return "", faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
		}
		if fileLen <= offset {
			return "", nil
		}
		if length == 0 || length > fileLen-offset {
			length = fileLen - offset
		}
		b := make([]byte, length)
		n, err := f.ReadAt(b, offset)
		if err != nil {
			log.WithField("error", err).Errorf("Fail to read log file %s at offset %d with length %d", name, offset, length)
			return "", faults.NewFault(faults.Failed, "FAILED")
		}
		return string(b[:n]), nil
	}

}

// ReadTailLog tails current log file
func (l *FileLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	if l.backups <= 0 {
		return "", 0, false, faults.NewFault(faults.NoFile, "NO_FILE")
	}

	l.locker.Lock()
	defer l.locker.Unlock()

	name := l.name
	if l.fileNameWithTimestamp {
		name = l.currentLogFileName
	}

	// open the file
	f, err := os.Open(name)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "name": name}).Errorf("Fail to open log file %s", name)
		return "", 0, false, faults.NewFault(faults.Failed, "Failed")
	}

	defer f.Close()

	// get the length of file
	statInfo, err := f.Stat()
	if err != nil {
		log.WithFields(log.Fields{"error": err, "name": name}).Errorf("Fail to get file info for %s", name)
		return "", 0, false, faults.NewFault(faults.Failed, "Failed")
	}

	fileLen := statInfo.Size()
	overflow := false

	if fileLen > offset+length {
		overflow = true
		offset = fileLen - 1
	}

	if offset+length > fileLen {
		if offset > fileLen-1 {
			length = 0
		}
		offset = fileLen - length
	}
	if offset < 0 {
		offset = 0
	}
	if length < 0 {
		length = 0
	}
	if length == 0 {
		return "", offset, overflow, nil
	}
	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", 0, false, faults.NewFault(faults.Failed, "Failed")
	}

	return string(b[:n]), offset + int64(n), overflow, nil

}

// Write overrides function in io.Writer. Write log message to the file
func (l *FileLogger) Write(p []byte) (int, error) {
	l.locker.Lock()
	defer l.locker.Unlock()

	l.logEventEmitter.emitLogEvent(string(p))

	if l.backups <= 0 {
		return 0, faults.NewFault(faults.NoFile, "NO_FILE")
	}

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

// Close ChanLogger — safe to call multiple times
func (l *ChanLogger) Close() error {
	l.once.Do(func() {
		close(l.channel)
	})
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
		writer: os.Stdout,
	}
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
		writer: os.Stderr,
	}
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

func NewMemoryLogger(n int, logEventEmitter LogEventEmitter) *MemoryLogger {
	return &MemoryLogger{
		logs:            ring.New(n),
		logEventEmitter: logEventEmitter,
	}
}

func (ml *MemoryLogger) Write(p []byte) (n int, err error) {
	ml.lock.Lock()
	defer ml.lock.Unlock()
	ml.logs.Value = string(p)
	ml.logs = ml.logs.Next()
	ml.logEventEmitter.emitLogEvent(string(p))
	return len(p), nil
}

func (ml *MemoryLogger) Close() error {
	return nil
}

func (ml *MemoryLogger) SetPid(pid int) {
	//NOTHING TO DO
}

func (ml *MemoryLogger) ReadLog(offset int64, length int64) (string, error) {
	if offset < 0 && length != 0 {
		return "", faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
	}
	if offset >= 0 && length < 0 {
		return "", faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
	}

	ml.lock.Lock()
	defer ml.lock.Unlock()
	var logs string = ""
	ml.logs.Do(func(p interface{}) {
		if p != nil {
			logs = logs + "\n" + p.(string)
		}
	})
	if offset > 0 && offset >= int64(len(logs)) {
		return "", errors.New("offset out of range")
	}
	if length <= 0 {
		return logs, nil
	}
	end := offset + length
	if end > int64(len(logs)) {
		end = int64(len(logs))
	}
	return logs[offset:end], nil
}

func (ml *MemoryLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	ml.lock.Lock()
	defer ml.lock.Unlock()
	var logs []string = make([]string, 0)
	ml.logs.Do(func(p interface{}) {
		if p != nil {
			logs = append(logs, p.(string))
		}
	})
	if offset > 0 && offset >= int64(len(logs)) {
		return "", offset, true, nil
	}
	end := offset + length
	if end > int64(len(logs)) {
		end = int64(len(logs))
	}
	return strings.Join(logs[offset:end], ""), end, end == int64(len(logs)), nil
}

func (ml *MemoryLogger) ClearCurLogFile() error {
	ml.lock.Lock()
	defer ml.lock.Unlock()
	ml.logs = ring.New(ml.logs.Len())
	return nil
}

func (ml *MemoryLogger) ClearAllLogFile() error {
	ml.lock.Lock()
	defer ml.lock.Unlock()
	ml.logs = ring.New(ml.logs.Len())
	return nil
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
func NewLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, fileNameWithTimestamp bool, props map[string]string, logEventEmitter LogEventEmitter) Logger {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for i, f := range files {
		var lr Logger
		if i == 0 {
			lr = createLogger(programName, f, locker, maxBytes, backups, fileNameWithTimestamp, props, logEventEmitter)
		} else {
			lr = createLogger(programName, f, NewNullLocker(), maxBytes, backups, fileNameWithTimestamp, props, NewNullLogEventEmitter())
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

func createLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, fileNameWithTimestamp bool, props map[string]string, logEventEmitter LogEventEmitter) Logger {
	switch logFile {
	case "/dev/stdout":
		return NewStdoutLogger(logEventEmitter)
	case "/dev/stderr":
		return NewStderrLogger(logEventEmitter)
	case "/dev/null":
		return NewNullLogger(logEventEmitter)
	case "syslog":
		return NewSysLogger(programName, props, logEventEmitter)
	case "memory":
		return NewMemoryLogger(1000, logEventEmitter)
	case "AUTO":
		return NewMemoryLogger(1000, logEventEmitter)
	default:
		if strings.HasPrefix(logFile, "syslog") {
			fields := strings.Split(logFile, "@")
			fields[0] = strings.TrimSpace(fields[0])
			fields[1] = strings.TrimSpace(fields[1])
			if len(fields) == 2 && fields[0] == "syslog" {
				return NewRemoteSysLogger(programName, fields[1], props, logEventEmitter)
			}
		}
		if len(logFile) > 0 {
			return NewFileLogger(logFile, maxBytes, backups, fileNameWithTimestamp, logEventEmitter, locker)
		}
		return NewNullLogger(logEventEmitter)

	}

}

type ReformatLog struct {
	output io.Writer
}

func NewReformatLog(output io.Writer) *ReformatLog {
	return &ReformatLog{output: output}
}

func (rl *ReformatLog) Write(p []byte) (n int, err error) {
	if len(p) == 0 || p[0] == '{' {
		return rl.output.Write(p)
	}
	message := string(p)

	timeStartPos := strings.Index(message, "time=\"")
	if timeStartPos != -1 {
		timeEndPos := strings.Index(message[timeStartPos+6:], "\"")
		if timeEndPos != -1 {
			timeEndPos += timeStartPos + 6
			message = message[:timeStartPos] + message[timeStartPos+6:timeEndPos] + message[timeEndPos+1:]
		}
	}

	levelStartPos := strings.Index(message, "level=")
	if levelStartPos != -1 {
		levelEndPos := strings.Index(message[levelStartPos+6:], " ")
		if levelEndPos != -1 {
			levelEndPos += levelStartPos + 6
			message = message[:levelStartPos] + "[" + message[levelStartPos+6:levelEndPos] + "] " + message[levelEndPos+1:]
		}
	}

	rl.output.Write([]byte(message))
	return len(p), nil

}
