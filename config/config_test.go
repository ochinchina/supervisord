package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
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
	config, err := parse([]byte("[program:test]\ncommand=/bin/ls"))
	if err != nil {
		t.Error("Fail to parse program")
		return
	}

	progs := config.GetPrograms()

	if len(progs) != 1 || config.GetProgram("test") == nil || config.GetProgram("app") != nil {
		t.Error("Fail to parse the test program")
	}
}

func TestGetBoolValueFromConfig(t *testing.T) {
	config, _ := parse([]byte("[program:test]\na=true\nb=false\n"))
	entry := config.GetProgram("test")
	if entry.GetBool("a", false) == false || entry.GetBool("b", true) == true || entry.GetBool("c", false) != false {
		t.Error("Fail to get boolean value")
	}
}

func TestGetIntValueFromConfig(t *testing.T) {
	config, _ := parse([]byte("[program:test]\na=1\nb=2\n"))
	entry := config.GetProgram("test")
	if entry.GetInt("a", 0) == 0 || entry.GetInt("b", 0) == 0 || entry.GetInt("c", 9) != 9 {
		t.Error("Fail to get integer value")
	}
}

func TestGetStringValueFromConfig(t *testing.T) {
	config, _ := parse([]byte("[program:test]\na=test\nb=hello\n"))
	entry := config.GetProgram("test")
	if entry.GetString("a", "") != "test" || entry.GetString("b", "") != "hello" || entry.GetString("c", "") != "" {
		t.Error("Fail to get string value")
	}
}

func TestGetEnvValueFromConfig(t *testing.T) {
	config, _ := parse([]byte("[program:test]\na=A=\"env1\",B=env2"))
	entry := config.GetProgram("test")
	envs := entry.GetEnv("a")
	if len(envs) != 2 || envs[0] != "A=env1" || envs[1] != "B=env2" {
		t.Error("Fail to get env value")
	}

	config, _ = parse([]byte("[program:test]\na=A=env1,B=\"env2\""))
	entry = config.GetProgram("test")
	envs = entry.GetEnv("a")
	if len(envs) != 2 || envs[0] != "A=env1" || envs[1] != "B=env2" {
		t.Error("Fail to get env value")
	}

}

func TestGetBytesFromConfig(t *testing.T) {
	config, _ := parse([]byte("[program:test]\nA=1024\nB=2KB\nC=3MB\nD=4GB\nE=test"))
	entry := config.GetProgram("test")

	if entry.GetBytes("A", 0) != 1024 || entry.GetBytes("B", 0) != 2048 || entry.GetBytes("C", 0) != 3*1024*1024 || entry.GetBytes("D", 0) != 4*1024*1024*1024 || entry.GetBytes("E", 0) != 0 || entry.GetBytes("F", -1) != -1 {
		t.Error("Fail to get bytes")
	}

}

func TestGetUnitHttpServer(t *testing.T) {
	config, _ := parse([]byte("[program:test]\nA=1024\nB=2KB\nC=3MB\nD=4GB\nE=test\n[unix_http_server]\n"))

	entry, ok := config.GetUnixHttpServer()

	if !ok || entry == nil {
		t.Error("Fail to get the unix_http_server")
	}

	if entry.GetProgramName() != "" {
		t.Error("there should be no program name in unix_http_server")
	}
}

func TestProgramInGroup(t *testing.T) {
	config, _ := parse([]byte("[program:test1]\nA=123\n[group:test]\nprograms=test1,test2\n[program:test2]\nB=hello\n[program:test3]\nC=tt"))
	if config.GetProgram("test1").Group != "test" { //|| config.GetProgram( "test2" ).Group != "test" || config.GetProgram( "test3" ).Group == "test" {
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

	ioutil.WriteFile(filepath.Join(dir, "file1"), []byte("[program:cat]\ncommand=pwd\nA=abc\n[include]\nfiles=*.conf"), os.ModePerm)
	ioutil.WriteFile(filepath.Join(dir, "file2.conf"), []byte("[program:ls]\ncommand=ls\n"), os.ModePerm)

	fmt.Println(filepath.Join(dir, "file1"))
	config := NewConfig(filepath.Join(dir, "file1"))
	config.Load()

	os.RemoveAll(filepath.Join(dir))

	entry := config.GetProgram("ls")

	if entry == nil {
		t.Error("fail to include section test")
	}

}

func TestDefaultParams(t *testing.T) {
	config, _ := parse([]byte("[program:test]\nautorestart=true\ntest=1\n[program-default]\ncommand=/usr/bin/ls\nrestart=true\nautorestart=false"))
	entry := config.GetProgram("test")
	if entry.GetString("command", "") != "/usr/bin/ls" {
		t.Error("fail to get command of program")
	}
	if entry.GetString("restart", "") != "true" {
		t.Error("Fail to get restart value")
	}

	if entry.GetInt("test", 0) != 1 {
		t.Error("Fail to get test value")
	}
	if entry.GetString("autorestart", "") != "true" {
		t.Error("autorestart value should be true")
	}

}
