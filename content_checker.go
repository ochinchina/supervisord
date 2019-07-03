package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type ContentChecker interface {
	Check() bool
}

type BaseChecker struct {
	data     string
	includes []string
	//timeout in second
	timeoutTime   time.Time
	notifyChannel chan string
}

func NewBaseChecker(includes []string, timeout int) *BaseChecker {
	return &BaseChecker{data: "",
		includes:      includes,
		timeoutTime:   time.Now().Add(time.Duration(timeout) * time.Second),
		notifyChannel: make(chan string, 1)}
}

func (bc *BaseChecker) Write(b []byte) (int, error) {
	bc.notifyChannel <- string(b)
	return len(b), nil
}

func (bc *BaseChecker) isReady() bool {
	find_all := true
	for _, include := range bc.includes {
		if strings.Index(bc.data, include) == -1 {
			find_all = false
			break
		}
	}
	return find_all
}
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

type ScriptChecker struct {
	args []string
}

func NewScriptChecker(args []string) *ScriptChecker {
	return &ScriptChecker{args: args}
}

func (sc *ScriptChecker) Check() bool {
	cmd := exec.Command(sc.args[0])
	if len(sc.args) > 1 {
		cmd.Args = sc.args
	}
	err := cmd.Run()
	return err == nil && cmd.ProcessState != nil && cmd.ProcessState.Success()
}

type TcpChecker struct {
	host        string
	port        int
	conn        net.Conn
	baseChecker *BaseChecker
}

func NewTcpChecker(host string, port int, includes []string, timeout int) *TcpChecker {
	checker := &TcpChecker{host: host,
		port:        port,
		baseChecker: NewBaseChecker(includes, timeout)}
	checker.start()
	return checker
}

func (tc *TcpChecker) start() {
	go func() {
		b := make([]byte, 1024)
		var err error
		for {
			tc.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", tc.host, tc.port))
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

func (tc *TcpChecker) Check() bool {
	ret := tc.baseChecker.Check()
	if tc.conn != nil {
		tc.conn.Close()
	}
	return ret
}

type HttpChecker struct {
	url         string
	timeoutTime time.Time
}

func NewHttpChecker(url string, timeout int) *HttpChecker {
	return &HttpChecker{url: url,
		timeoutTime: time.Now().Add(time.Duration(timeout) * time.Second)}
}

func (hc *HttpChecker) Check() bool {
	for {
		if hc.timeoutTime.After(time.Now()) {
			resp, err := http.Get(hc.url)
			if err == nil {
				return resp.StatusCode >= 200 && resp.StatusCode < 300
			}
		}
	}
}
