package events

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// EventSysVersion the event system version
	EventSysVersion = "3.0"

	// ProcCommonBeginStr the process communication begin
	ProcCommonBeginStr = "<!--XSUPERVISOR:BEGIN-->"

	// ProcCommonEndStr the process communication end
	ProcCommonEndStr = "<!--XSUPERVISOR:END-->"
)

// Event the event interface definition
type Event interface {
	GetSerial() uint64
	GetType() string
	GetBody() string
}

// BaseEvent the base event, all other events should inherit this BaseEvent to implement the Event interface
type BaseEvent struct {
	serial    uint64
	eventType string
}

// GetSerial returns serial number of event
func (be *BaseEvent) GetSerial() uint64 {
	return be.serial
}

// GetType returns type of given event
func (be *BaseEvent) GetType() string {
	return be.eventType
}

// EventListenerManager manage the event listeners
type EventListenerManager struct {
	// mapping between the event listener name and the listener
	namedListeners map[string]*EventListener
	// mapping between the event name and the event listeners
	eventListeners map[string]map[*EventListener]bool
}

// EventPoolSerial manage the event serial generation
type EventPoolSerial struct {
	sync.Mutex
	poolserial map[string]uint64
}

// NewEventPoolSerial creates new EventPoolSerial object
func NewEventPoolSerial() *EventPoolSerial {
	return &EventPoolSerial{poolserial: make(map[string]uint64)}
}

func (eps *EventPoolSerial) nextSerial(pool string) uint64 {
	eps.Lock()
	defer eps.Unlock()

	r, ok := eps.poolserial[pool]
	if !ok {
		r = 1
	}
	eps.poolserial[pool] = r + 1
	return r
}

// EventListener the event listener object
type EventListener struct {
	pool       string
	server     string
	cond       *sync.Cond
	events     *list.List
	stdin      *bufio.Reader
	stdout     io.Writer
	bufferSize int
}

// NewEventListener creates NewEventListener object
func NewEventListener(pool string,
	server string,
	stdin io.Reader,
	stdout io.Writer,
	bufferSize int) *EventListener {
	evtListener := &EventListener{pool: pool,
		server:     server,
		cond:       sync.NewCond(new(sync.Mutex)),
		events:     list.New(),
		stdin:      bufio.NewReader(stdin),
		stdout:     stdout,
		bufferSize: bufferSize}
	evtListener.start()
	return evtListener
}

func (el *EventListener) getFirstEvent() ([]byte, bool) {
	el.cond.L.Lock()

	defer el.cond.L.Unlock()

	for el.events.Len() <= 0 {
		el.cond.Wait()
	}

	if el.events.Len() > 0 {
		elem := el.events.Front()
		value := elem.Value
		b, ok := value.([]byte)
		return b, ok
	}
	return nil, false
}

func (el *EventListener) removeFirstEvent() {
	el.cond.L.Lock()
	defer el.cond.L.Unlock()
	if el.events.Len() > 0 {
		el.events.Remove(el.events.Front())
	}
}

func (el *EventListener) start() {
	go func() {
		for {
			// read if it is ready
			err := el.waitForReady()
			if err != nil {
				log.WithFields(log.Fields{"eventListener": el.pool}).Warn("fail to read from event listener, the event listener may exit")
				break
			}
			for {
				if b, ok := el.getFirstEvent(); ok {
					_, err := el.stdout.Write(b)
					if err != nil {
						log.WithFields(log.Fields{"eventListener": el.pool}).Warn("fail to send event")
						break
					}
					result, err := el.readResult()
					if err != nil {
						log.WithFields(log.Fields{"eventListener": el.pool}).Warn("fail to read result")
						break
					}
					if result == "OK" { // remove the event if succeed
						log.WithFields(log.Fields{"eventListener": el.pool}).Info("succeed to send the event")
						el.removeFirstEvent()
						break
					} else if result == "FAIL" {
						log.WithFields(log.Fields{"eventListener": el.pool}).Warn("fail to send the event")
						break
					} else {
						log.WithFields(log.Fields{"eventListener": el.pool, "result": result}).Warn("unknown result from listener")
					}
				}
			}
		}
	}()
}

func (el *EventListener) waitForReady() error {
	log.Debug("start to check if event listener program is ready")
	for {
		line, err := el.stdin.ReadString('\n')
		if err != nil {
			return err
		}
		if line == "READY\n" {
			log.WithFields(log.Fields{"eventListener": el.pool}).Debug("the event listener is ready")
			return nil
		}
	}
}

func (el *EventListener) readResult() (string, error) {
	s, err := el.stdin.ReadString('\n')
	if err != nil {
		return s, err
	}
	fields := strings.Fields(s)
	if len(fields) == 2 && fields[0] == "RESULT" {
		// try to get the length of result
		n, err := strconv.Atoi(fields[1])
		if err != nil {
			// return if fail to get the length
			return "", err
		}
		if n < 0 {
			return "", fmt.Errorf("Fail to read the result because the result bytes is less than 0")
		}
		// read n bytes
		b := make([]byte, n)
		for i := 0; i < n; i++ {
			b[i], err = el.stdin.ReadByte()
			if err != nil {
				return "", err
			}
		}
		// ok, get the n bytes
		return string(b), nil
	}
	return "", fmt.Errorf("Fail to read the result")
}

// HandleEvent handles emitted event
func (el *EventListener) HandleEvent(event Event) {
	encodedEvent := el.encodeEvent(event)
	el.cond.L.Lock()
	defer el.cond.L.Unlock()
	if el.events.Len() <= el.bufferSize {
		el.events.PushBack(encodedEvent)
		el.cond.Signal()
	} else {
		log.WithFields(log.Fields{"eventListener": el.pool}).Error("events reaches the bufferSize, discard the events")
	}
}

func (el *EventListener) encodeEvent(event Event) []byte {
	body := []byte(event.GetBody())

	// header
	s := fmt.Sprintf("ver:%s server:%s serial:%d pool:%s poolserial:%d eventname:%s len:%d\n",
		EventSysVersion,
		el.server,
		event.GetSerial(),
		el.pool,
		eventPoolSerial.nextSerial(el.pool),
		event.GetType(),
		len(body))
	// write the header & body to buffer
	r := bytes.NewBuffer([]byte(s))
	r.Write(body)

	return r.Bytes()
}

var eventTypeDerives = map[string][]string{
	"PROCESS_STATE_STARTING":           {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_RUNNING":            {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_BACKOFF":            {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_STOPPING":           {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_EXITED":             {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_STOPPED":            {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_FATAL":              {"EVENT", "PROCESS_STATE"},
	"PROCESS_STATE_UNKNOWN":            {"EVENT", "PROCESS_STATE"},
	"REMOTE_COMMUNICATION":             {"EVENT"},
	"PROCESS_LOG_STDOUT":               {"EVENT", "PROCESS_LOG"},
	"PROCESS_LOG_STDERR":               {"EVENT", "PROCESS_LOG"},
	"PROCESS_COMMUNICATION_STDOUT":     {"EVENT", "PROCESS_COMMUNICATION"},
	"PROCESS_COMMUNICATION_STDERR":     {"EVENT", "PROCESS_COMMUNICATION"},
	"SUPERVISOR_STATE_CHANGE_RUNNING":  {"EVENT", "SUPERVISOR_STATE_CHANGE"},
	"SUPERVISOR_STATE_CHANGE_STOPPING": {"EVENT", "SUPERVISOR_STATE_CHANGE"},
	"TICK_5":                           {"EVENT", "TICK"},
	"TICK_60":                          {"EVENT", "TICK"},
	"TICK_3600":                        {"EVENT", "TICK"},
	"PROCESS_GROUP_ADDED":              {"EVENT", "PROCESS_GROUP"},
	"PROCESS_GROUP_REMOVED":            {"EVENT", "PROCESS_GROUP"}}
var eventSerial uint64
var eventListenerManager = NewEventListenerManager()
var eventPoolSerial = NewEventPoolSerial()

func init() {
	startTickTimer()
}

func startTickTimer() {
	tickConfigs := map[string]int64{"TICK_5": 5,
		"TICK_60":   60,
		"TICK_3600": 3600}

	// start a Tick timer
	go func() {
		lastTickSlice := make(map[string]int64)

		c := time.Tick(1 * time.Second)
		for now := range c {
			for tickType, period := range tickConfigs {
				timeSlice := now.Unix() / period
				lastTimeSlice, ok := lastTickSlice[tickType]
				if !ok {
					lastTickSlice[tickType] = timeSlice
				} else if lastTimeSlice != timeSlice {
					lastTickSlice[tickType] = timeSlice
					EmitEvent(NewTickEvent(tickType, now.Unix()))
				}
			}
		}
	}()
}

func nextEventSerial() uint64 {
	return atomic.AddUint64(&eventSerial, 1)
}

// NewEventListenerManager creates EventListenerManager object
func NewEventListenerManager() *EventListenerManager {
	return &EventListenerManager{namedListeners: make(map[string]*EventListener),
		eventListeners: make(map[string]map[*EventListener]bool)}
}

func (em *EventListenerManager) registerEventListener(eventListenerName string,
	events []string,
	listener *EventListener) {

	em.namedListeners[eventListenerName] = listener
	allEvents := make(map[string]bool)
	for _, event := range events {
		for k, values := range eventTypeDerives {
			if event == k { // if it is a final event
				allEvents[k] = true
			} else { // if it is an abstract event, add all its derived events
				for _, val := range values {
					if val == event {
						allEvents[k] = true
					}
				}
			}
		}
	}
	for event := range allEvents {
		log.WithFields(log.Fields{"eventListener": eventListenerName, "event": event}).Info("register event listener")
		if _, ok := em.eventListeners[event]; !ok {
			em.eventListeners[event] = make(map[*EventListener]bool)
		}
		em.eventListeners[event][listener] = true
	}
}

// RegisterEventListener registers event listener to accept the emitted events
func RegisterEventListener(eventListenerName string,
	events []string,
	listener *EventListener) {
	eventListenerManager.registerEventListener(eventListenerName, events, listener)
}

func (em *EventListenerManager) unregisterEventListener(eventListenerName string) *EventListener {
	listener, ok := em.namedListeners[eventListenerName]
	if ok {
		delete(em.namedListeners, eventListenerName)
		for event, listeners := range em.eventListeners {
			if _, ok = listeners[listener]; ok {
				log.WithFields(log.Fields{"eventListener": eventListenerName, "event": event}).Info("unregister event listener")
			}

			delete(listeners, listener)
		}
		return listener
	}
	return nil
}

// UnregisterEventListener unregisters event listener by its name
func UnregisterEventListener(eventListenerName string) *EventListener {
	return eventListenerManager.unregisterEventListener(eventListenerName)
}

// EmitEvent emits event to all listeners managed by this manager
func (em *EventListenerManager) EmitEvent(event Event) {
	listeners, ok := em.eventListeners[event.GetType()]
	if ok {
		log.WithFields(log.Fields{"event": event.GetType()}).Info("process event")
		for listener := range listeners {
			log.WithFields(log.Fields{"eventListener": listener.pool, "event": event.GetType()}).Info("receive event on listener")
			listener.HandleEvent(event)
		}
	}
}

// RemoteCommunicationEvent remote communication event definition
type RemoteCommunicationEvent struct {
	BaseEvent
	typ  string
	data string
}

// NewRemoteCommunicationEvent creates new RemoteCommunicationEvent object
func NewRemoteCommunicationEvent(typ string, data string) *RemoteCommunicationEvent {
	r := &RemoteCommunicationEvent{typ: typ, data: data}
	r.eventType = "REMOTE_COMMUNICATION"
	r.serial = nextEventSerial()
	return r
}

// GetBody returns event' body
func (r *RemoteCommunicationEvent) GetBody() string {
	return fmt.Sprintf("type:%s\n%s", r.typ, r.data)
}

// ProcCommEvent process communication event definition
type ProcCommEvent struct {
	BaseEvent
	processName string
	groupName   string
	pid         int
	data        string
}

// NewProcCommEvent creates new ProcCommEvent object
func NewProcCommEvent(eventType string,
	procName string,
	groupName string,
	pid int,
	data string) *ProcCommEvent {
	return &ProcCommEvent{BaseEvent: BaseEvent{eventType: eventType, serial: nextEventSerial()},
		processName: procName,
		groupName:   groupName,
		pid:         pid,
		data:        data}
}

// GetBody returns process communication event' body
func (p *ProcCommEvent) GetBody() string {
	return fmt.Sprintf("processname:%s groupname:%s pid:%d\n%s", p.processName, p.groupName, p.pid, p.data)
}

// EmitEvent emits event to default event listener manager
func EmitEvent(event Event) {
	eventListenerManager.EmitEvent(event)
}

// TickEvent the tick event definition
type TickEvent struct {
	BaseEvent
	when int64
}

// NewTickEvent creates new periodical TickEvent object
func NewTickEvent(tickType string, when int64) *TickEvent {
	r := &TickEvent{when: when}
	r.eventType = tickType
	r.serial = nextEventSerial()
	return r
}

// GetBody returns TickEvent' body
func (te *TickEvent) GetBody() string {
	return fmt.Sprintf("when:%d", te.when)
}

// ProcCommEventCapture process communication event capture
type ProcCommEventCapture struct {
	reader          io.Reader
	captureMaxBytes int
	stdType         string
	procName        string
	groupName       string
	pid             int
	eventBuffer     string
	eventBeginPos   int
}

// NewProcCommEventCapture creates new ProcCommEventCapture object
func NewProcCommEventCapture(reader io.Reader,
	captureMaxBytes int,
	stdType string,
	procName string,
	groupName string) *ProcCommEventCapture {
	pec := &ProcCommEventCapture{reader: reader,
		captureMaxBytes: captureMaxBytes,
		stdType:         stdType,
		procName:        procName,
		groupName:       groupName,
		pid:             -1,
		eventBuffer:     "",
		eventBeginPos:   -1}
	pec.startCapture()
	return pec
}

// SetPid sets pid of the program
func (pec *ProcCommEventCapture) SetPid(pid int) {
	pec.pid = pid
}

func (pec *ProcCommEventCapture) startCapture() {
	go func() {
		buf := make([]byte, 10240)
		for {
			n, err := pec.reader.Read(buf)
			if err != nil {
				break
			}
			pec.eventBuffer += string(buf[0:n])
			for {
				event := pec.captureEvent()
				if event == nil {
					break
				}
				EmitEvent(event)
			}
		}
	}()
}

func (pec *ProcCommEventCapture) captureEvent() Event {
	pec.findBeginStr()
	endPos := pec.findEndStr()
	if endPos == -1 {
		return nil
	}
	data := pec.eventBuffer[pec.eventBeginPos+len(ProcCommonBeginStr) : endPos]
	pec.eventBuffer = pec.eventBuffer[endPos+len(ProcCommonEndStr):]
	pec.eventBeginPos = -1
	return NewProcCommEvent(pec.stdType,
		pec.procName,
		pec.groupName,
		pec.pid,
		data)
}

func (pec *ProcCommEventCapture) findBeginStr() {
	if pec.eventBeginPos == -1 {
		pec.eventBeginPos = strings.Index(pec.eventBuffer, ProcCommonBeginStr)
		if pec.eventBeginPos == -1 {
			// remove some string
			n := len(pec.eventBuffer)
			if n > len(ProcCommonBeginStr) {
				pec.eventBuffer = pec.eventBuffer[n-len(ProcCommonBeginStr):]
			}
		}
	}
}

func (pec *ProcCommEventCapture) findEndStr() int {
	if pec.eventBeginPos == -1 {
		return -1
	}
	endPos := strings.Index(pec.eventBuffer, ProcCommonEndStr)
	if endPos == -1 {
		if len(pec.eventBuffer) > pec.captureMaxBytes {
			log.WithFields(log.Fields{"program": pec.procName}).Warn("The capture buffer is overflow, discard the content")
			pec.eventBeginPos = -1
			pec.eventBuffer = ""
		}
	}
	return endPos
}

// ProcessStateEvent process state event definition
type ProcessStateEvent struct {
	BaseEvent
	processName string
	groupName   string
	fromState   string
	tries       int
	expected    int
	pid         int
}

// CreateProcessStartingEvent emits create process starting event
func CreateProcessStartingEvent(process string,
	group string,
	fromState string,
	tries int) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     tries,
		expected:  -1,
		pid:       0}
	r.eventType = "PROCESS_STATE_STARTING"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessRunningEvent emits create process running event
func CreateProcessRunningEvent(process string,
	group string,
	fromState string,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     -1,
		expected:  -1,
		pid:       pid}
	r.eventType = "PROCESS_STATE_RUNNING"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessBackoffEvent emits create process backoff event
func CreateProcessBackoffEvent(process string,
	group string,
	fromState string,
	tries int) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     tries,
		expected:  -1,
		pid:       0}
	r.eventType = "PROCESS_STATE_BACKOFF"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessStoppingEvent emits create process stopping event
func CreateProcessStoppingEvent(process string,
	group string,
	fromState string,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     -1,
		expected:  -1,
		pid:       pid}
	r.eventType = "PROCESS_STATE_STOPPING"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessExitedEvent emits create process exited event
func CreateProcessExitedEvent(process string,
	group string,
	fromState string,
	expected int,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     -1,
		expected:  expected,
		pid:       pid}
	r.eventType = "PROCESS_STATE_EXITED"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessStoppedEvent emits create process stopped event
func CreateProcessStoppedEvent(process string,
	group string,
	fromState string,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     -1,
		expected:  -1,
		pid:       pid}
	r.eventType = "PROCESS_STATE_STOPPED"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessFatalEvent emits create process fatal error event
func CreateProcessFatalEvent(process string,
	group string,
	fromState string) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     -1,
		expected:  -1,
		pid:       0}
	r.eventType = "PROCESS_STATE_FATAL"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessUnknownEvent emits create process state unknown event
func CreateProcessUnknownEvent(process string,
	group string,
	fromState string) *ProcessStateEvent {
	r := &ProcessStateEvent{processName: process,
		groupName: group,
		fromState: fromState,
		tries:     -1,
		expected:  -1,
		pid:       0}
	r.eventType = "PROCESS_STATE_UNKNOWN"
	r.serial = nextEventSerial()
	return r
}

// GetBody returns body of process state event
func (pse *ProcessStateEvent) GetBody() string {
	body := fmt.Sprintf("processname:%s groupname:%s from_state:%s", pse.processName, pse.groupName, pse.fromState)
	if pse.tries >= 0 {
		body = fmt.Sprintf("%s tries:%d", body, pse.tries)
	}

	if pse.expected != -1 {
		body = fmt.Sprintf("%s expected:%d", body, pse.expected)
	}

	if pse.pid != 0 {
		body = fmt.Sprintf("%s pid:%d", body, pse.pid)
	}
	return body
}

// SupervisorStateChangeEvent supervisor state change event
type SupervisorStateChangeEvent struct {
	BaseEvent
}

// GetBody returns body of supervisor state change event
func (s *SupervisorStateChangeEvent) GetBody() string {
	return ""
}

// CreateSupervisorStateChangeRunning creates SupervisorStateChangeEvent object
func CreateSupervisorStateChangeRunning() *SupervisorStateChangeEvent {
	r := &SupervisorStateChangeEvent{}
	r.eventType = "SUPERVISOR_STATE_CHANGE_RUNNING"
	r.serial = nextEventSerial()
	return r
}

func createSupervisorStateChangeStopping() *SupervisorStateChangeEvent {
	r := &SupervisorStateChangeEvent{}
	r.eventType = "SUPERVISOR_STATE_CHANGE_STOPPING"
	r.serial = nextEventSerial()
	return r
}

// ProcessLogEvent process log event definition
type ProcessLogEvent struct {
	BaseEvent
	processName string
	groupName   string
	pid         int
	data        string
}

// GetBody returns body of process log event
func (pe *ProcessLogEvent) GetBody() string {
	return fmt.Sprintf("processname:%s groupname:%s pid:%d\n%s",
		pe.processName,
		pe.groupName,
		pe.pid,
		pe.data)
}

// CreateProcessLogStdoutEvent emits create process stdout log event
func CreateProcessLogStdoutEvent(processName string,
	groupName string,
	pid int,
	data string) *ProcessLogEvent {
	r := &ProcessLogEvent{processName: processName,
		groupName: groupName,
		pid:       pid,
		data:      data}
	r.eventType = "PROCESS_LOG_STDOUT"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessLogStderrEvent emits create process stderr log event
func CreateProcessLogStderrEvent(processName string,
	groupName string,
	pid int,
	data string) *ProcessLogEvent {
	r := &ProcessLogEvent{processName: processName,
		groupName: groupName,
		pid:       pid,
		data:      data}
	r.eventType = "PROCESS_LOG_STDERR"
	r.serial = nextEventSerial()
	return r
}

// ProcessGroupEvent the process group event definition
type ProcessGroupEvent struct {
	BaseEvent
	groupName string
}

// GetBody returns body of process group event
func (pe *ProcessGroupEvent) GetBody() string {
	return fmt.Sprintf("groupname:%s", pe.groupName)
}

// CreateProcessGroupAddedEvent emits create process group added event
func CreateProcessGroupAddedEvent(groupName string) *ProcessGroupEvent {
	r := &ProcessGroupEvent{groupName: groupName}

	r.eventType = "PROCESS_GROUP_ADDED"
	r.serial = nextEventSerial()
	return r
}

// CreateProcessGroupRemovedEvent emits create process group removed event
func CreateProcessGroupRemovedEvent(groupName string) *ProcessGroupEvent {
	r := &ProcessGroupEvent{groupName: groupName}

	r.eventType = "PROCESS_GROUP_REMOVED"
	r.serial = nextEventSerial()
	return r
}
