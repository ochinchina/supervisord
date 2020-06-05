package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLogFile(t *testing.T) {
	files := splitLogFile(" test1.log, /dev/stdout, test2.log ")
	assert.ElementsMatch(t, []string{"test1.log", "/dev/stdout", "test2.log"}, files)
}
