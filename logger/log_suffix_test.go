//go:build linux

package logger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A timestamp suffix must not be added to a log file that is not a regular
// file: "/dev/fd/1.<timestamp>" cannot be opened, so the program's output was
// lost and its next write failed with SIGPIPE.
func TestTimestampSuffixSkippedForNonRegularFile(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	path := fmt.Sprintf("/proc/self/fd/%d", w.Fd())
	l := NewLogger("test", path, NewNullLocker(), 1<<20, 10, true, map[string]string{}, NewNullLogEventEmitter())
	defer l.Close()
	if _, err := l.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write to %s: %v", path, err)
	}

	lines := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(r).ReadString('\n')
		lines <- line
	}()
	select {
	case line := <-lines:
		if line != "hello\n" {
			t.Fatalf("read %q from the pipe, want %q", line, "hello\n")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("nothing written to the pipe")
	}
}

// Regular files still get the suffix when it is asked for.
func TestTimestampSuffixAppliedToRegularFile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "program.log")
	l := NewLogger("test", name, NewNullLocker(), 1<<20, 10, true, map[string]string{}, NewNullLogEventEmitter())
	defer l.Close()
	if _, err := l.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(name + ".*")
	if len(files) != 1 {
		t.Fatalf("got timestamped files %v, want exactly one", files)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatalf("unsuffixed %s exists (err %v), want only the timestamped file", name, err)
	}
}
