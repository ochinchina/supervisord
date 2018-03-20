package logger

import (
	"errors"
	"fmt"
	"github.com/ochinchina/supervisord/events"
	"github.com/ochinchina/supervisord/faults"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

//implements io.Writer interface

type Logger interface {
	io.WriteCloser
	SetPid(pid int)
	ReadLog(offset int64, length int64) (string, error)
	ReadTailLog(offset int64, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

type LogEventEmitter interface {
	emitLogEvent(data string)
}

type FileLogger struct {
	name            string
	maxSize         int64
	backups         int
	curRotate       int
	fileSize        int64
	file            *os.File
	logEventEmitter LogEventEmitter
	locker          sync.Locker
}

type SysLogger struct {
	NullLogger
	logWriter       io.WriteCloser
	logEventEmitter LogEventEmitter
}

type NullLogger struct {
	logEventEmitter LogEventEmitter
}

type NullLocker struct {
}

func NewFileLogger(name string, maxSize int64, backups int, logEventEmitter LogEventEmitter, locker sync.Locker) *FileLogger {
	logger := &FileLogger{name: name,
		maxSize:         maxSize,
		backups:         backups,
		curRotate:       -1,
		fileSize:        0,
		file:            nil,
		logEventEmitter: logEventEmitter,
		locker:          locker}
	logger.updateLatestLog()
	return logger
}

func (l *FileLogger) SetPid(pid int) {
	//NOTHING TO DO
}

// return the next log file name
func (l *FileLogger) nextLogFile() {
	l.curRotate++
	if l.curRotate >= l.backups {
		l.curRotate = 0
	}
}

func (l *FileLogger) updateLatestLog() {
	dir := path.Dir(l.name)
	files, err := ioutil.ReadDir(dir)
	baseName := path.Base(l.name)

	if err != nil {
		l.curRotate = 0
	} else {
		//find all the rotate files
		var latestFile os.FileInfo
		latestNum := -1
		for _, fileInfo := range files {
			if !fileInfo.IsDir() && strings.HasPrefix(fileInfo.Name(), baseName+".") {
				n, err := strconv.Atoi(fileInfo.Name()[len(baseName)+1:])
				if err == nil && n >= 0 && n < l.backups {
					if latestFile == nil || latestFile.ModTime().Before(fileInfo.ModTime()) {
						latestFile = fileInfo
						latestNum = n
					}
				}
			}
		}
		l.curRotate = latestNum
		if latestFile != nil {
			l.fileSize = latestFile.Size()
		} else {
			l.fileSize = int64(0)
		}
		if l.fileSize >= l.maxSize || latestFile == nil {
			l.nextLogFile()
			l.openFile(true)
		} else {
			l.openFile(false)
		}
	}
}

// open the file and truncate the file if trunc is true
func (l *FileLogger) openFile(trunc bool) error {
	if l.file != nil {
		l.file.Close()
	}
	var err error
	fileName := l.GetCurrentLogFile()
	if trunc {
		l.file, err = os.Create(fileName)
	} else {
		l.file, err = os.OpenFile(fileName, os.O_RDWR|os.O_APPEND, 0666)
	}
	return err
}

// get the name of current log file
func (l *FileLogger) GetCurrentLogFile() string {
	return l.getLogFileName(l.curRotate)
}

// get the name of previous log file
func (l *FileLogger) GetPrevLogFile() string {
	i := (l.curRotate - 1 + l.backups) % l.backups

	return l.getLogFileName(i)
}

func (l *FileLogger) getLogFileName(index int) string {
	return fmt.Sprintf("%s.%d", l.name, index)
}

// clear the current log file contents
func (l *FileLogger) ClearCurLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	return l.openFile(true)
}

func (l *FileLogger) ClearAllLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	for i := 0; i < l.backups && i <= l.curRotate; i++ {
		logFile := l.getLogFileName(i)
		err := os.Remove(logFile)
		if err != nil {
			return faults.NewFault(faults.FAILED, err.Error())
		}
	}
	l.curRotate = 0
	err := l.openFile(true)
	if err != nil {
		return faults.NewFault(faults.FAILED, err.Error())
	}
	return nil
}

func (l *FileLogger) ReadLog(offset int64, length int64) (string, error) {
	if offset < 0 && length != 0 {
		return "", faults.NewFault(faults.BAD_ARGUMENTS, "BAD_ARGUMENTS")
	}
	if offset >= 0 && length < 0 {
		return "", faults.NewFault(faults.BAD_ARGUMENTS, "BAD_ARGUMENTS")
	}

	l.locker.Lock()
	defer l.locker.Unlock()
	f, err := os.Open(l.GetCurrentLogFile())

	if err != nil {
		return "", faults.NewFault(faults.FAILED, "FAILED")
	}
	defer f.Close()

	//check the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", faults.NewFault(faults.FAILED, "FAILED")
	}

	fileLen := statInfo.Size()

	if offset < 0 { //offset < 0 && length == 0
		offset = fileLen + offset
		if offset < 0 {
			offset = 0
		}
		length = fileLen - offset
	} else if length == 0 { //offset >= 0 && length == 0
		if offset > fileLen {
			return "", nil
		}
		length = fileLen - offset
	} else { //offset >= 0 && length > 0

		//if the offset exceeds the length of file
		if offset >= fileLen {
			return "", nil
		}

		//compute actual bytes should be read

		if offset+length > fileLen {
			length = fileLen - offset
		}
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", faults.NewFault(faults.FAILED, "FAILED")
	}
	return string(b[:n]), nil
}

func (l *FileLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	if offset < 0 {
		return "", offset, false, fmt.Errorf("offset should not be less than 0")
	}
	if length < 0 {
		return "", offset, false, fmt.Errorf("length should be not be less than 0")
	}
	l.locker.Lock()
	defer l.locker.Unlock()

	//open the file
	f, err := os.Open(l.GetCurrentLogFile())
	if err != nil {
		return "", 0, false, err
	}

	defer f.Close()

	//get the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}

	fileLen := statInfo.Size()

	//check if offset exceeds the length of file
	if offset >= fileLen {
		return "", fileLen, true, nil
	}

	//get the length
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

// Override the function in io.Writer
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
		fileInfo, errStat := os.Stat(fmt.Sprintf("%s.%d", l.name, l.curRotate))
		if errStat == nil {
			l.fileSize = fileInfo.Size()
		} else {
			return n, errStat
		}
	}
	if l.fileSize >= l.maxSize {
		l.nextLogFile()
		l.openFile(true)
	}
	return n, err
}

func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (sl *SysLogger) Write(b []byte) (int, error) {
	sl.logEventEmitter.emitLogEvent(string(b))
	if sl.logWriter == nil {
		return 0, errors.New("not connect to syslog server")
	}
	return sl.logWriter.Write(b)
}

func (sl *SysLogger) Close() error {
	if sl.logWriter == nil {
		return errors.New("not connect to syslog server")
	}
	return sl.logWriter.Close()
}
func NewNullLogger(logEventEmitter LogEventEmitter) *NullLogger {
	return &NullLogger{logEventEmitter: logEventEmitter}
}

func (l *NullLogger) SetPid(pid int) {
	//NOTHING TO DO
}

func (l *NullLogger) Write(p []byte) (int, error) {
	l.logEventEmitter.emitLogEvent(string(p))
	return len(p), nil
}

func (l *NullLogger) Close() error {
	return nil
}

func (l *NullLogger) ReadLog(offset int64, length int64) (string, error) {
	return "", faults.NewFault(faults.NO_FILE, "NO_FILE")
}

func (l *NullLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NO_FILE, "NO_FILE")
}

func (l *NullLogger) ClearCurLogFile() error {
	return fmt.Errorf("No log")
}

func (l *NullLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NO_FILE, "NO_FILE")
}

func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

func (l *NullLocker) Lock() {
}

func (l *NullLocker) Unlock() {
}

type StdLogger struct {
	NullLogger
	logEventEmitter LogEventEmitter
	writer          io.Writer
}

func NewStdoutLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{logEventEmitter: logEventEmitter,
		writer: os.Stdout}
}

func (l *StdLogger) Write(p []byte) (int, error) {
	n, err := l.writer.Write(p)
	if err != nil {
		l.logEventEmitter.emitLogEvent(string(p))
	}
	return n, err
}

func NewStderrLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{logEventEmitter: logEventEmitter,
		writer: os.Stderr}
}

type LogCaptureLogger struct {
	underlineLogger        Logger
	procCommEventCapWriter io.Writer
	procCommEventCapture   *events.ProcCommEventCapture
}

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

func (l *LogCaptureLogger) SetPid(pid int) {
	l.procCommEventCapture.SetPid(pid)
}

func (l *LogCaptureLogger) Write(p []byte) (int, error) {
	l.procCommEventCapWriter.Write(p)
	return l.underlineLogger.Write(p)
}

func (l *LogCaptureLogger) Close() error {
	return l.underlineLogger.Close()
}

func (l *LogCaptureLogger) ReadLog(offset int64, length int64) (string, error) {
	return l.underlineLogger.ReadLog(offset, length)
}

func (l *LogCaptureLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return l.underlineLogger.ReadTailLog(offset, length)
}

func (l *LogCaptureLogger) ClearCurLogFile() error {
	return l.underlineLogger.ClearCurLogFile()
}

func (l *LogCaptureLogger) ClearAllLogFile() error {
	return l.underlineLogger.ClearAllLogFile()
}

type NullLogEventEmitter struct {
}

func NewNullLogEventEmitter() *NullLogEventEmitter {
	return &NullLogEventEmitter{}
}

func (ne *NullLogEventEmitter) emitLogEvent(data string) {
}

type StdLogEventEmitter struct {
	Type         string
	process_name string
	group_name   string
	pidFunc      func() int
}

func NewStdoutLogEventEmitter(process_name string, group_name string, procPidFunc func() int) *StdLogEventEmitter {
	return &StdLogEventEmitter{Type: "stdout",
		process_name: process_name,
		group_name:   group_name,
		pidFunc:      procPidFunc}
}

func NewStderrLogEventEmitter(process_name string, group_name string, procPidFunc func() int) *StdLogEventEmitter {
	return &StdLogEventEmitter{Type: "stderr",
		process_name: process_name,
		group_name:   group_name,
		pidFunc:      procPidFunc}
}

func (se *StdLogEventEmitter) emitLogEvent(data string) {
	if se.Type == "stdout" {
		events.EmitEvent(events.CreateProcessLogStdoutEvent(se.process_name, se.group_name, se.pidFunc(), data))
	} else {
		events.EmitEvent(events.CreateProcessLogStderrEvent(se.process_name, se.group_name, se.pidFunc(), data))
	}
}
