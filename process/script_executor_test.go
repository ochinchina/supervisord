package process

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewScriptExecutor(t *testing.T) {
	se := NewScriptExecutor("echo hello")
	if se == nil {
		t.Fatal("expected non-nil ScriptExecutor")
	}
	if se.script != "echo hello" {
		t.Errorf("expected script %q, got %q", "echo hello", se.script)
	}
}

func TestFileExists(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-fileexists-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if !fileExists(tmpFile.Name()) {
		t.Error("expected fileExists to return true for existing file")
	}
	if fileExists(filepath.Join(os.TempDir(), "nonexistent-file-12345")) {
		t.Error("expected fileExists to return false for non-existing file")
	}
}

func TestExecuteLocal(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo hello"
	} else {
		cmd = "echo hello"
	}
	se := NewScriptExecutor(cmd)
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteLocalWithScriptPrefix(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "script://cmd /c echo hello"
	} else {
		cmd = "script://echo hello"
	}
	se := NewScriptExecutor(cmd)
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteHTTP_GET(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer ts.Close()

	se := NewScriptExecutor(ts.URL + " -H Content-Type:application/json")
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteHTTP_POST(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	se := NewScriptExecutor(ts.URL + " -d testdata")
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteHTTP_PostWithDataOption(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	se := NewScriptExecutor(ts.URL + " --data mypayload")
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteHTTP_PostWithFileData(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-data-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.WriteString("file-content")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	se := NewScriptExecutor(ts.URL + " -d @" + tmpFile.Name())
	err = se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteHTTP_WithHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom"); got != "value1" {
			t.Errorf("expected header X-Custom=value1, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer-token" {
			t.Errorf("expected header Authorization=Bearer-token, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Header values cannot contain spaces since the script is split by strings.Fields
	se := NewScriptExecutor(ts.URL + " -H X-Custom:value1 --header Authorization:Bearer-token -d x")
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteHTTP_URLOnlyNoParams(t *testing.T) {
	se := NewScriptExecutor("http://127.0.0.1:1/noop")
	// URL only with no extra fields — the code returns nil without making a request
	err := se.Execute()
	if err != nil {
		t.Errorf("expected no error for URL-only script, got %v", err)
	}
}

func TestExecuteHTTP_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	se := NewScriptExecutor(ts.URL + " -d data")
	err := se.Execute()
	// current implementation does not treat HTTP status errors as errors
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteTCP_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	se := NewScriptExecutor("tcp://" + ln.Addr().String())
	err = se.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteTCP_Failure(t *testing.T) {
	se := NewScriptExecutor("tcp://127.0.0.1:1")
	err := se.Execute()
	if err == nil {
		t.Error("expected error for unreachable TCP address")
	}
}

func TestParseHttpParameters_OptionsAndHeaders(t *testing.T) {
	se := NewScriptExecutor("")
	params := []string{"-key", "keyfile", "--cert", "certfile", "-H", "Accept:text/html", "extra"}
	options, paramsList, headers := se.parseHttpParameters(params)

	if options["key"] != "keyfile" {
		t.Errorf("expected key=keyfile, got %q", options["key"])
	}
	if options["cert"] != "certfile" {
		t.Errorf("expected cert=certfile, got %q", options["cert"])
	}
	if headers["Accept"] != "text/html" {
		t.Errorf("expected Accept=text/html, got %q", headers["Accept"])
	}
	if len(paramsList) != 1 || paramsList[0] != "extra" {
		t.Errorf("expected paramsList=[extra], got %v", paramsList)
	}
}

func TestParseHttpParameters_Empty(t *testing.T) {
	se := NewScriptExecutor("")
	options, paramsList, headers := se.parseHttpParameters([]string{})
	if len(options) != 0 || len(paramsList) != 0 || len(headers) != 0 {
		t.Error("expected all empty maps/slices for empty params")
	}
}

func TestLoadData_Empty(t *testing.T) {
	se := NewScriptExecutor("")
	data := se.loadData(map[string]string{})
	if len(data) != 0 {
		t.Errorf("expected empty data, got %q", data)
	}
}

func TestLoadData_InlineData(t *testing.T) {
	se := NewScriptExecutor("")
	data := se.loadData(map[string]string{"data": "inline-payload"})
	if string(data) != "inline-payload" {
		t.Errorf("expected %q, got %q", "inline-payload", string(data))
	}
}

func TestLoadData_ShortFlag(t *testing.T) {
	se := NewScriptExecutor("")
	data := se.loadData(map[string]string{"d": "short-payload"})
	if string(data) != "short-payload" {
		t.Errorf("expected %q, got %q", "short-payload", string(data))
	}
}

func TestLoadData_FromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-loaddata-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.WriteString("file-payload")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	se := NewScriptExecutor("")
	data := se.loadData(map[string]string{"data": "@" + tmpFile.Name()})
	if string(data) != "file-payload" {
		t.Errorf("expected %q, got %q", "file-payload", string(data))
	}
}

func TestLoadData_FromNonExistentFile(t *testing.T) {
	se := NewScriptExecutor("")
	data := se.loadData(map[string]string{"data": "@/nonexistent/file/path"})
	if len(data) != 0 {
		t.Errorf("expected empty data for non-existent file, got %q", data)
	}
}

func TestExecuteRouting(t *testing.T) {
	tests := []struct {
		name   string
		script string
		prefix string
	}{
		{"http", "http://example.com -d x", "http://"},
		{"https", "https://example.com -d x", "https://"},
		{"tcp", "tcp://127.0.0.1:1", "tcp://"},
		{"local", "echo hello", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := NewScriptExecutor(tt.script)
			// Just verify the executor was created with correct script
			if se.script != tt.script {
				t.Errorf("expected script %q, got %q", tt.script, se.script)
			}
		})
	}
}
