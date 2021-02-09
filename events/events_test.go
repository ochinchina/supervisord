package events

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEventSerial(t *testing.T) {
	v1 := nextEventSerial()
	v2 := nextEventSerial()
	if v2 < v1 {
		t.Error("Fail to get next serial")
	}
}

func TestEventPoolSerial(t *testing.T) {
	val := eventPoolSerial.nextSerial("test1")
	if val != 1 {
		t.Error("Fail to get next serial")
	}

	val = eventPoolSerial.nextSerial("test1")
	if val != 2 {
		t.Error("Fail to get next serial")
	}

	val = eventPoolSerial.nextSerial("test2")
	if val != 1 {
		t.Error("Fail to get next serial")
	}

}

func readEvent(reader *bufio.Reader) (string, string) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return "", ""
	}
	tmp := strings.Split(header[0:len(header)-1], ":")
	len, _ := strconv.Atoi(tmp[len(tmp)-1])
	b := make([]byte, len)
	io.ReadFull(reader, b)
	return header, string(b)
}

func TestEventListener(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	reader := bufio.NewReader(r1)

	listener := NewEventListener("pool-1",
		"supervisor",
		r2,
		w1,
		10)
	eventListenerManager.registerEventListener("pool-1",
		[]string{"REMOTE_COMMUNICATION"},
		listener)
	EmitEvent(NewRemoteCommunicationEvent("type-1", "this is a remote communication event test"))
	fmt.Printf("start to write READY\n")
	w2.Write([]byte("READY\n"))
	_, body := readEvent(reader)
	if body != "type:type-1\nthis is a remote communication event test" {
		t.Error("The body is not expect")
	}
	w2.Write([]byte("RESULT 4\nFAIL"))
	w2.Write([]byte("READY\n"))
	_, body = readEvent(reader)
	if body != "type:type-1\nthis is a remote communication event test" {
		t.Error("The body is not expect")
	}
	w2.Write([]byte("RESULT 2\nOK"))
	time.Sleep(2 * time.Second)
	w2.Close()
	r2.Close()
	r1.Close()
	w1.Close()

	eventListenerManager.unregisterEventListener("pool-1")
}

func TestProcCommEventCapture(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	reader := bufio.NewReader(r1)

	captureReader, captureWriter := io.Pipe()
	eventCapture := NewProcCommEventCapture(captureReader,
		10240,
		"PROCESS_COMMUNICATION_STDOUT",
		"proc-1",
		"group-1")
	eventCapture.SetPid(99)
	listener := NewEventListener("pool-1",
		"supervisor",
		r2,
		w1,
		10)
	eventListenerManager.registerEventListener("pool-1",
		[]string{"PROCESS_COMMUNICATION"},
		listener)
	w2.Write([]byte("READY\n"))
	captureWriter.Write([]byte(`this is unuseful information, seems it is very 
	long and not useful, just used for testing purpose.
	let's input more unuseful information, ok.....
	haha...<!--XSUPERVISOR:BEGIN-->this is a proc event test<!--XSUPERVISOR:END--> also
	add some other unuseful`))
	_, body := readEvent(reader)
	expectBody := "processname:proc-1 groupname:group-1 pid:99\nthis is a proc event test"
	if body != expectBody {
		t.Error("Fail to get the process communication event")
	}
	w2.Close()
	r2.Close()
	r1.Close()
	w1.Close()
}

func TestProcessStartingEvent(t *testing.T) {
	event := CreateProcessStartingEvent("proc-1", "group-1", "STOPPED", 0)
	if event.GetType() != "PROCESS_STATE_STARTING" {
		t.Error("Fail to creating the process starting event")
	}
	fmt.Printf( "%s\n", event.GetBody() )
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:STOPPED tries:0" {
		t.Error("Fail to encode the process starting event")
	}
}

func TestProcessRunningEvent(t *testing.T) {
	event := CreateProcessRunningEvent("proc-1", "group-1", "STARTING", 2766)
	if event.GetType() != "PROCESS_STATE_RUNNING" {
		t.Error("Fail to creating the process running event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:STARTING pid:2766" {
		t.Error("Fail to encode the process running event")
	}
}

func TestProcessBackoffEvent(t *testing.T) {
	event := CreateProcessBackoffEvent("proc-1", "group-1", "STARTING", 1)
	if event.GetType() != "PROCESS_STATE_BACKOFF" {
		t.Error("Fail to creating the process backoff event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:STARTING tries:1" {
		t.Error("Fail to encode the process backoff event")
	}
}

func TestProcessStoppingEvent(t *testing.T) {
	event := CreateProcessStoppingEvent("proc-1", "group-1", "STARTING", 2766)
	if event.GetType() != "PROCESS_STATE_STOPPING" {
		t.Error("Fail to creating the process stopping event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:STARTING pid:2766" {
		t.Error("Fail to encode the process stopping event")
	}
}

func TestProcessExitedEvent(t *testing.T) {
	event := CreateProcessExitedEvent("proc-1", "group-1", "RUNNING", 1, 2766)
	if event.GetType() != "PROCESS_STATE_EXITED" {
		t.Error("Fail to creating the process exited event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:RUNNING expected:1 pid:2766" {
		t.Error("Fail to encode the process exited event")
	}
}

func TestProcessStoppedEvent(t *testing.T) {
	event := CreateProcessStoppedEvent("proc-1", "group-1", "STOPPING", 2766)
	if event.GetType() != "PROCESS_STATE_STOPPED" {
		t.Error("Fail to creating the process stopped event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:STOPPING pid:2766" {
		t.Error("Fail to encode the process stopped event")
	}
}

func TestProcessFatalEvent(t *testing.T) {
	event := CreateProcessFatalEvent("proc-1", "group-1", "BACKOFF")
	if event.GetType() != "PROCESS_STATE_FATAL" {
		t.Error("Fail to creating the process fatal event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:BACKOFF" {
		t.Error("Fail to encode the process fatal event")
	}
}

func TestProcessUnknownEvent(t *testing.T) {
	event := CreateProcessUnknownEvent("proc-1", "group-1", "BACKOFF")
	if event.GetType() != "PROCESS_STATE_UNKNOWN" {
		t.Error("Fail to creating the process unknown event")
	}
	if event.GetBody() != "processname:proc-1 groupname:group-1 from_state:BACKOFF" {
		t.Error("Fail to encode the process unknown event")
	}
}
