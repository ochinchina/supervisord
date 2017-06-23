package main

import (
	"fmt"
	"bufio"
	"io"
	"strings"
	"strconv"
	"testing"
	"time"
)

func TestEventSerial(t *testing.T) {
	val := nextEventSerial()
	if val != 1 {
		t.Error("Fail to get next serial")
	}
	
	val = nextEventSerial()
	if val != 2 {
		t.Error("Fail to get next serial")
	}
}

func TestEventPoolSerial(t *testing.T) {
	val := eventPoolSerial.nextSerial( "test1")
	if val != 1 {
		t.Error("Fail to get next serial")
	}
	
	val = eventPoolSerial.nextSerial( "test1")
	if val != 2 {
		t.Error("Fail to get next serial")
	}
	
	val = eventPoolSerial.nextSerial( "test2")
	if val != 1 {
		t.Error("Fail to get next serial")
	}
		
}

func readEvent( reader *bufio.Reader ) (string, string) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return "", ""
	} else {
		tmp := strings.Split( header[0:len(header)-1], ":")
		len, _ := strconv.Atoi(tmp[ len( tmp ) - 1 ] )
		b := make([]byte, len )
		io.ReadFull( reader, b )
		return header, string( b )
	}
}
func TestEventListener(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2:= io.Pipe()
	reader := bufio.NewReader( r1 )
	
	listener := NewEventListener( "pool-1", 
				"supervisor",
				r2,
				w1,
				10)
	eventListenerManager.registerEventListener( "pool-1",
				[]string{"REMOTE_COMMUNICATION"}, 
				listener )
	emitEvent( NewRemoteCommunicationEvent( "type-1", "this is a remote communication event test") )
	fmt.Printf( "start to write READY\n")
	w2.Write( []byte("READY\n") )
	_, body := readEvent( reader )
	if body != "type:type-1\nthis is a remote communication event test" {
		t.Error("The body is not expect")
	}
	w2.Write( []byte("RESULT 4\nFAIL") )
	w2.Write( []byte("READY\n") )
	_, body = readEvent( reader )
	if body != "type:type-1\nthis is a remote communication event test" {
		t.Error("The body is not expect")
	}
	w2.Write( []byte("RESULT 2\nOK") )
	time.Sleep(2 * time.Second)
	w2.Close()
	r2.Close()
	r1.Close()
	w1.Close()
	
	eventListenerManager.unregisterEventListener( "pool-1" )
}

func TestProcCommEventCapture(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2:= io.Pipe()
	reader := bufio.NewReader( r1 )
	
	capture_reader, capture_writer := io.Pipe()
	eventCapture := NewProcCommEventCapture( capture_reader,
				10240,
				"PROCESS_COMMUNICATION_STDOUT",
				"proc-1",
				"group-1" )
	eventCapture.SetPid( 99 )
	listener := NewEventListener( "pool-1", 
				"supervisor",
				r2,
				w1,
				10)
	eventListenerManager.registerEventListener( "pool-1",
				[]string{"PROCESS_COMMUNICATION"}, 
				listener )
	w2.Write( []byte("READY\n") )
	capture_writer.Write( []byte(`this is unuseful information, seems it is very 
	long and not useful, just used for testing purpose.
	let's input more unuseful information, ok.....
	haha...<!--XSUPERVISOR:BEGIN-->this is a proc event test<!--XSUPERVISOR:END--> also
	add some other unuseful`) )
	_, body := readEvent( reader )
	expect_body := "processname:proc-1 groupname:group-1 pid:99\nthis is a proc event test"
	if body != expect_body {
		t.Error("Fail to get the process communication event")
	}
	w2.Close()
	r2.Close()
	r1.Close()
	w1.Close()
	
}
	