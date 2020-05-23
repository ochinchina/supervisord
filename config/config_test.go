package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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
	if err != nil {
		t.Error("Fail to parse program")
		return
	}

	progs := config.Programs()

	if len(progs) != 1 || config.GetProgram("test") == nil || config.GetProgram("app") != nil {
		t.Error("Fail to parse the test program")
	}
}

func TestUnixHttpServer(t *testing.T) {
	config, _ := parse([]byte("[program.test]\nA=1024\nB=2KB\nC=3MB\nD=4GB\nE=test\n[unix_http_server]\nfile=/foo.bar"))

	entry := config.UnixHTTPServer
	assert.NotNil(t, entry)
	assert.Equal(t, "/foo.bar", entry.File)
}

func TestInetHttpServer(t *testing.T) {
	config, _ := parse([]byte("[program.test]\nA=1024\nB=2KB\nC=3MB\nD=4GB\nE=test\n[inet_http_server]\nport=9898"))

	entry := config.InetHTTPServer
	assert.NotNil(t, entry)
	assert.Equal(t, "9898", entry.Port)
}

func TestProgramInGroup(t *testing.T) {
	config, _ := parse([]byte("[program.test1]\nA=123\n[group.test]\nprograms=test1,test2\n[program.test2]\nB=hello\n[program.test3]\nC=tt"))
	if config.GetProgram("test1").Group != "test" {
		t.Error("fail to test the program in a group")
	}
}

func TestToRegex(t *testing.T) {
	pattern := toRegexp("/an/absolute/*.conf")
	matched, err := regexp.MatchString(pattern, "/an/absolute/ab.conf")
	if !matched || err != nil {
		t.Error("fail to match the file")
	}

	matched, err = regexp.MatchString(pattern, "/an/absolute/abconf")

	if matched && err == nil {
		t.Error("fail to match the file")
	}

	pattern = toRegexp("/an/absolute/??.conf")
	matched, err = regexp.MatchString(pattern, "/an/absolute/ab.conf")
	if !matched || err != nil {
		t.Error("fail to match the file")
	}

	matched, err = regexp.MatchString(pattern, "/an/absolute/abconf")
	if matched && err == nil {
		t.Error("fail to match the file")
	}

	matched, err = regexp.MatchString(pattern, "/an/absolute/abc.conf")
	if matched && err == nil {
		t.Error("fail to match the file")
	}
}

func TestConfigWithInclude(t *testing.T) {
	dir, _ := ioutil.TempDir("", "tmp")

	ioutil.WriteFile(filepath.Join(dir, "file1"), []byte("[program.cat]\ncommand=pwd\nA=abc\n[include]\nfiles=*.conf"), os.ModePerm)
	ioutil.WriteFile(filepath.Join(dir, "file2.conf"), []byte("[program.ls]\ncommand=ls\n"), os.ModePerm)

	config := NewConfig(filepath.Join(dir, "file1"))
	config.Load()

	os.RemoveAll(filepath.Join(dir))

	entry := config.GetProgram("ls")

	if entry == nil {
		t.Error("fail to include section test")
	}
}

func TestDefaultParams(t *testing.T) {
	config, _ := parse([]byte("[program.test]\nautorestart=true\ntest=1\n[program]\ncommand=/usr/bin/ls\nstartretries=10\nautorestart=false"))
	entry := config.GetProgram("test")
	assert.Equal(t, "/usr/bin/ls", entry.Command)
	assert.True(t, entry.AutoStart)
	assert.Equal(t, entry.StartRetries, 10)
}
