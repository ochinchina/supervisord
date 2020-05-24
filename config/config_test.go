package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTmpFile() (string, error) {
	f, err := ioutil.TempFile("", "tmp")
	if err == nil {
		f.Close()
		return f.Name(), err
	}
	return "", err
}

func saveToTmpFile(b []byte) (string, error) {
	f, err := createTmpFile()
	if err != nil {
		return "", err
	}

	ioutil.WriteFile(f, b, os.ModePerm)

	return f, nil
}

func parse(b []byte) (*Config, error) {
	fileName, err := saveToTmpFile(b)
	if err != nil {
		return nil, err
	}
	config := NewConfig(fileName)
	_, err = config.Load()

	if err != nil {
		return nil, err
	}
	os.Remove(fileName)
	return config, nil
}

func TestProgramConfig(t *testing.T) {
	config, err := parse([]byte("[program.test]\ncommand=/bin/ls"))
	assert.NoError(t, err)

	progs := config.Programs()
	assert.Len(t, progs, 1)
	assert.NotNil(t, config.GetProgram("test"))
	assert.Nil(t, config.GetProgram("app"))
}

func TestHttpServer(t *testing.T) {
	config, _ := parse([]byte("[program.test]\nA=1024\nB=2KB\nC=3MB\nD=4GB\nE=test\n[http_server]\nport=9898"))

	entry := config.HttpServer
	assert.NotNil(t, entry)
	assert.Equal(t, "9898", entry.Port)
}

func TestProgramInGroup(t *testing.T) {
	config, _ := parse([]byte("[program.test1]\nA=123\n[group.test]\nprograms=test1,test2\n[program.test2]\nB=hello\n[program.test3]\nC=tt"))
	assert.NotNil(t, config.GetProgram("test1"))
	assert.Equal(t, "test", config.GetProgram("test1").Group)
}

func TestDefaultParams(t *testing.T) {
	config, _ := parse([]byte("[program.test]\nautorestart=true\ntest=1\n[program]\ncommand=/usr/bin/ls\nstartretries=10\nautorestart=false"))
	entry := config.GetProgram("test")
	assert.NotNil(t, entry)
	assert.Equal(t, "/usr/bin/ls", entry.Command)
	assert.True(t, entry.AutoStart)
	assert.Equal(t, entry.StartRetries, 10)
}
