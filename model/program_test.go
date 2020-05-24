package model_test

import (
	"testing"
	"time"

	"github.com/stuartcarnie/gopm/model"

	"github.com/creasty/defaults"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestProgramDefaults(t *testing.T) {
	var got model.Program
	_ = defaults.Set(&got)
	exp := model.Program{
		ExitCodes:             []int{0, 2},
		AutoStart:             true,
		Priority:              999,
		StartRetries:          3,
		StopWaitSeconds:       model.Duration(10 * time.Second),
		StartSeconds:          model.Duration(1 * time.Second),
		RestartFilePattern:    "*",
		StdoutLogFile:         "/dev/null",
		StdoutLogfileBackups:  10,
		StdoutLogFileMaxBytes: 50 * 1024 * 1024,
		StderrLogFile:         "/dev/null",
		StderrLogfileBackups:  10,
		StderrLogFileMaxBytes: 50 * 1024 * 1024,
	}
	assert.Equal(t, exp, got)
}

func TestProgram_UnmarshalYAML(t *testing.T) {
	data := `
name: hello world!
stop_wait_seconds: 5s
`
	var p model.Program
	err := yaml.Unmarshal([]byte(data), &p)
	assert.NoError(t, err)
}
