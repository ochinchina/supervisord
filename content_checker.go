package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ContentChecker defines check interface
type ContentChecker interface {
	Check() bool
}

// BaseChecker basic implementation of ContentChecker
type BaseChecker struct {
	data     string
	includes []string
	// timeout in second
	timeoutTime   time.Time
	notifyChannel chan string
}

// NewBaseChecker creates BaseChecker object
func NewBaseChecker(includes []string, timeout int) *BaseChecker {
	return &BaseChecker{data: "",
		includes:      includes,
		timeoutTime:   time.Now().Add(time.Duration(timeout) * time.Second),
		notifyChannel: make(chan string, 1)}
}

// Write data to the checker
func (bc *BaseChecker) Write(b []byte) (int, error) {
	bc.notifyChannel <- string(b)
	return len(b), nil
}

func (bc *BaseChecker) isReady() bool {
	for _, include := range bc.includes {
		if !strings.Contains(bc.data, include) {
			return false
		}
	}
	return true
}

// Check content of the input data
func (bc *BaseChecker) Check() bool {
	d := bc.timeoutTime.Sub(time.Now())
	if d < 0 {
		return false
	}
	timeoutSignal := time.After(d)

	for {
		select {
		case data := <-bc.notifyChannel:
			bc.data = bc.data + data
			if bc.isReady() {
				return true
			}
		case <-timeoutSignal:
			return false
		}
	}
}

// ScriptChecker implements ContentChecker by calling external script
type ScriptChecker struct {
	args []string
}

// NewScriptChecker creates ScriptChecker object
func NewScriptChecker(args []string) *ScriptChecker {
	return &ScriptChecker{args: args}
}

// Check return code of the script. If return code is 0, check is successful
func (sc *ScriptChecker) Check() bool {
	cmd := exec.Command(sc.args[0])
	if len(sc.args) > 1 {
		cmd.Args = sc.args
	}
	err := cmd.Run()
	return err == nil && cmd.ProcessState != nil && cmd.ProcessState.Success()
}

// TCPChecker check by TCP protocol
type TCPChecker struct {
	host        string
	port        int
	conn        net.Conn
	baseChecker *BaseChecker
}

// NewTCPChecker creates TCPChecker object
func NewTCPChecker(host string, port int, includes []string, timeout int) *TCPChecker {
	checker := &TCPChecker{host: host,
		port:        port,
		baseChecker: NewBaseChecker(includes, timeout)}
	checker.start()
	return checker
}

func (tc *TCPChecker) start() {
	go func() {
		b := make([]byte, 1024)
		var err error
		for {
			tc.conn, err = net.Dial("tcp", net.JoinHostPort(tc.host, fmt.Sprintf("%d", tc.port)))
			if err == nil || tc.baseChecker.timeoutTime.Before(time.Now()) {
				break
			}
		}

		if err == nil {
			for {
				n, err := tc.conn.Read(b)
				if err != nil {
					break
				}
				tc.baseChecker.Write(b[0:n])
			}
		}
	}()
}

// Check if it is ready by reading the tcp data
func (tc *TCPChecker) Check() bool {
	ret := tc.baseChecker.Check()
	if tc.conn != nil {
		tc.conn.Close()
	}
	return ret
}

// HTTPChecker implements the ContentChecker by HTTP protocol
type HTTPChecker struct {
	url         string
	timeoutTime time.Time
}

// NewHTTPChecker creates HTTPChecker object
func NewHTTPChecker(url string, timeout int) *HTTPChecker {
	return &HTTPChecker{url: url,
		timeoutTime: time.Now().Add(time.Duration(timeout) * time.Second)}
}

// Check content of HTTP response
func (hc *HTTPChecker) Check() bool {
	for {
		if hc.timeoutTime.After(time.Now()) {
			resp, err := http.Get(hc.url)
			if err == nil {
				return resp.StatusCode >= 200 && resp.StatusCode < 300
			}
		}
	}
}
