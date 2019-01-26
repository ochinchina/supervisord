package logger

import (
	"fmt"
	"testing"
)

func TestWriteSingleLog(t *testing.T) {
	logger := NewFileLogger("test.log", int64(50), 2, NewNullLogEventEmitter(), NewNullLocker())
	for i := 0; i < 10; i++ {
		logger.Write([]byte(fmt.Sprintf("this is a test %d\n", i)))
	}
	logger.Close()
}

func TestSplitLogFile(t *testing.T) {
	files := splitLogFile(" test1.log, /dev/stdout, test2.log ")
	if len(files) != 3 {
		t.Error("Fail to split log file")
	}
	if files[0] != "test1.log" {
		t.Error("Fail to get first log file")
	}
	if files[1] != "/dev/stdout" {
		t.Error("Fail to get second log file")
	}
	if files[2] != "test2.log" {
		t.Error("Fail to get third log file")
	}

}
