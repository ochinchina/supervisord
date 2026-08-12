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

// TestHttpCheckTimeoutReturns verifies that Check() returns false within a
// reasonable time when no server is available. With the old code this test
// would hang forever after the timeout expired.
func TestHttpCheckTimeoutReturns(t *testing.T) {
	start := time.Now()
	checker := NewHTTPChecker("http://127.0.0.1:19991", 1)
	if checker.Check() {
		t.Error("expected false when no server is listening")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("Check() took %v, should return within ~1s of timeout", elapsed)
	}
}

// TestHttpCheckRetriesOnConnectionRefused verifies that Check() retries when
// the server is not yet listening (connection refused), and eventually succeeds
// once the server comes up. This also exercises the body-close path on success.
func TestHttpCheckRetriesOnConnectionRefused(t *testing.T) {
	// Grab a free port, then immediately release it so it's "not yet listening".
	tmp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := tmp.Addr().String()
	tmp.Close()

	// Start the real server after a short delay to force at least one retry.
	go func() {
		time.Sleep(400 * time.Millisecond)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return
		}
		http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}()

	checker := NewHTTPChecker("http://"+addr, 5)
	if !checker.Check() {
		t.Error("expected true after server became available")
	}
}
