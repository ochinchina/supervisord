package logger

import (
	"testing"
)

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
