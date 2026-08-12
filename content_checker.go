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
	data          strings.Builder
	includes      []string
	timeoutTime   time.Time
	notifyChannel chan string
}

// NewBaseChecker creates BaseChecker object
func NewBaseChecker(includes []string, timeout int) *BaseChecker {
	return &BaseChecker{
		includes:      includes,
		timeoutTime:   time.Now().Add(time.Duration(timeout) * time.Second),
		notifyChannel: make(chan string, 1),
	}
}

// Write data to the checker
func (bc *BaseChecker) Write(b []byte) (int, error) {
	bc.notifyChannel <- string(b)
	return len(b), nil
}

func (bc *BaseChecker) isReady() bool {
	s := bc.data.String()
	for _, include := range bc.includes {
		if !strings.Contains(s, include) {
			return false
		}
	}
	return true
}

// Check content of the input data
func (bc *BaseChecker) Check() bool {
	d := time.Until(bc.timeoutTime)
	if d < 0 {
		return false
	}
	timeoutSignal := time.After(d)

	for {
		select {
		case data := <-bc.notifyChannel:
			bc.data.WriteString(data)
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
	checker := &TCPChecker{
		host:        host,
		port:        port,
		baseChecker: NewBaseChecker(includes, timeout),
	}
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
			time.Sleep(500 * time.Millisecond)
		}

		if err == nil {
			for {
				n, err := tc.conn.Read(b)
				if err != nil {
					break
				}
				_, _ = tc.baseChecker.Write(b[0:n])
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
	return &HTTPChecker{
		url:         url,
		timeoutTime: time.Now().Add(time.Duration(timeout) * time.Second),
	}
}

// Check content of HTTP response. Returns true if a 2xx response is received
// before the timeout. Returns false if the timeout expires or the response
// is non-2xx.
func (hc *HTTPChecker) Check() bool {
	for {
		if time.Now().Before(hc.timeoutTime) {
			resp, err := http.Get(hc.url)
			if err == nil {
				resp.Body.Close()
				return resp.StatusCode >= 200 && resp.StatusCode < 300
			}
			time.Sleep(500 * time.Millisecond)
		} else {
			return false
		}
	}
}
