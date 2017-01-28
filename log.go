package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

//implements io.Writer interface
type Logger struct {
	name string
	maxSize int64
	backups int
	curRotate int
	fileSize int64
	file *os.File
	locker sync.Locker
}

type NullLogger struct {
}

type NullLocker struct {
}

func NewLogger( name string, maxSize int64, backups int, locker sync.Locker ) *Logger {
	logger := &Logger{ name: name,
			maxSize: maxSize,
			backups: backups,
			curRotate:-1,
			fileSize: 0,
			file: nil,
			locker: locker }
	logger.updateLatestLog()
	return logger
}

// return the next log file name
func (l *Logger) nextLogFile() {
	l.curRotate += 1
	if l.curRotate >= l.backups {
		l.curRotate = 0
	}
}

func (l *Logger)updateLatestLog() {
	dir := path.Dir( l.name )
	files, err := ioutil.ReadDir( dir )

	if err != nil {
		l.curRotate = 0
	} else {
		//find all the rotate files
		var latestFile os.FileInfo = nil
		latestNum := -1 
		for _, fileInfo := range files {
			if strings.HasPrefix( fileInfo.Name(), l.name + "." ) {
				n, err := strconv.Atoi( fileInfo.Name()[len( l.name ) + 1:] )
				if err == nil && n >= 0 && n < l.backups {
					if latestFile == nil || latestFile.ModTime().Before( fileInfo.ModTime() ) {
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
			l.openFile( true )
		} else {
			l.openFile( false )
		}
	}
}

func (l *Logger) openFile( trunc bool ) error {
	if l.file != nil {
		l.file.Close()
	}
	var err error = nil
	fileName :=  fmt.Sprintf( "%s.%d", l.name, l.curRotate )
	if trunc {
		l.file, err = os.Create( fileName)
	} else {
		l.file, err = os.OpenFile( fileName,  os.O_RDWR|os.O_APPEND, 0666 )
	}
	return err
}
// Override the function in io.Writer
func (l *Logger) Write(p []byte) (int, error){
	l.locker.Lock()
	defer l.locker.Unlock()

	n, err := l.file.Write( p )

	if err != nil {
		return n, err
	}
	 l.fileSize += int64(n)
	if l.fileSize >= l.maxSize {
		fileInfo, err := os.Stat( fmt.Sprintf( "%s.%d", l.name, l.curRotate ) )
		if err == nil {
			l.fileSize = fileInfo.Size()
		} else {
			return n, err
		}
	}
	if l.fileSize >= l.maxSize {
		l.nextLogFile()
		l.openFile( true )
	}
	return n, err
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	} else {
		return nil
	}
}

func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

func (l *NullLogger)Write( p []byte)( int, error ) {
	return len(p), nil
}

func (l* NullLogger) Close() error {
	return nil
}

func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

func (l *NullLocker) Lock() {
}

func (l *NullLocker) Unlock() {
}
