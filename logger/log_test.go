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
