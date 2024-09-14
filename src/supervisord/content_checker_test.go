package main

import (
	"net"
	"net/http"
	"testing"
	"time"
)

func TestBaseCheckOk(t *testing.T) {
	checker := NewBaseChecker([]string{"Hello", "world"}, 10)

	go func() {
		checker.Write([]byte("this is a world"))
		time.Sleep(2 * time.Second)
		checker.Write([]byte("Hello, how are you?"))
	}()
	if !checker.Check() {
		t.Fail()
	}
}

func TestBaseCheckFail(t *testing.T) {
	checker := NewBaseChecker([]string{"Hello", "world"}, 2)

	go func() {
		checker.Write([]byte("this is a world"))
	}()
	if checker.Check() {
		t.Fail()
	}
}

func TestTcpCheckOk(t *testing.T) {
	go func() {
		listener, err := net.Listen("tcp", ":8999")
		if err == nil {
			defer listener.Close()
			conn, err := listener.Accept()
			if err == nil {
				defer conn.Close()
				conn.Write([]byte("this is a world"))
				time.Sleep(3 * time.Second)
				conn.Write([]byte("Hello, how are you?"))
			}
		}
	}()
	checker := NewTCPChecker("127.0.0.1", 8999, []string{"Hello", "world"}, 10)
	if !checker.Check() {
		t.Fail()
	}
}

func TestTcpCheckFail(t *testing.T) {
	go func() {
		listener, err := net.Listen("tcp", ":8989")
		if err == nil {
			conn, err := listener.Accept()
			if err == nil {
				conn.Write([]byte("this is a world"))
				time.Sleep(3 * time.Second)
				conn.Close()
			}
			listener.Close()
		}
	}()
	checker := NewTCPChecker("127.0.0.1", 8989, []string{"Hello", "world"}, 2)
	if checker.Check() {
		t.Fail()
	}
}

func TestHttpCheckOk(t *testing.T) {
	go func() {
		listener, err := net.Listen("tcp", ":8999")
		if err == nil {

			http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer listener.Close()
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("this is an response"))
			}))

		}
	}()
	checker := NewHTTPChecker("http://127.0.0.1:8999", 2)
	if !checker.Check() {
		t.Fail()
	}
}

func TestHttpCheckFail(t *testing.T) {
	go func() {
		listener, err := net.Listen("tcp", ":8999")
		if err == nil {
			http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer listener.Close()
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("not found"))
			}))

		}
	}()
	checker := NewHTTPChecker("http://127.0.0.1:8999", 2)
	if checker.Check() {
		t.Fail()
	}
}
