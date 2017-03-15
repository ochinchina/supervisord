package main

import (
	"fmt"
	"testing"
)

func TestWriteSingleLog(t *testing.T) {
	logger := NewLogger("test.log", int64(50), 2)
	for i := 0; i < 10; i++ {
		logger.Write([]byte(fmt.Sprintf("this is a test %d\n", i)))
	}
	logger.Close()
}
