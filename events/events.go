package events

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	EVENT_SYS_VERSION     = "3.0"
	PROC_COMMON_BEGIN_STR = "<!--XSUPERVISOR:BEGIN-->"
	PROC_COMMON_END_STR   = "<!--XSUPERVISOR:END-->"
)

type Event interface {
	GetSerial() uint64
	GetType() string
	GetBody() string
}

type BaseEvent struct {
	serial    uint64
	eventType string
}

func (be *BaseEvent) GetSerial() uint64 {
	return be.serial
}

func (be *BaseEvent) GetType() string {
	return be.eventType
}

type EventListenerManager struct {
	//mapping between the event listener name and the listener
	namedListeners map[string]*EventListener
	//mapping between the event name and the event listeners
	eventListeners map[string]map[*EventListener]bool
}

type EventPoolSerial struct {
	sync.Mutex
	poolserial map[string]uint64
}

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

type EventListener struct {
	pool        string
	server      string
	cond        *sync.Cond
	events      *list.List
	stdin       *bufio.Reader
	stdout      io.Writer
	buffer_size int
}

func NewEventListener(pool string,
	server string,
	stdin io.Reader,
	stdout io.Writer,
	buffer_size int) *EventListener {
	evtListener := &EventListener{pool: pool,
		server:      server,
		cond:        sync.NewCond(new(sync.Mutex)),
		events:      list.New(),
		stdin:       bufio.NewReader(stdin),
		stdout:      stdout,
		buffer_size: buffer_size}
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
			//read if it is ready
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
					if result == "OK" { //remove the event if succeed
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
		//try to get the length of result
		n, err := strconv.Atoi(fields[1])
		if err != nil {
			//return if fail to get the length
			return "", err
		}
		if n < 0 {
			return "", fmt.Errorf("Fail to read the result because the result bytes is less than 0")
		}
		//read n bytes
		b := make([]byte, n)
		for i := 0; i < n; i++ {
			b[i], err = el.stdin.ReadByte()
			if err != nil {
				return "", err
			}
		}
		//ok, get the n bytes
		return string(b), nil
	}
	return "", fmt.Errorf("Fail to read the result")
}

func (el *EventListener) HandleEvent(event Event) {
	encodedEvent := el.encodeEvent(event)
	el.cond.L.Lock()
	defer el.cond.L.Unlock()
	if el.events.Len() <= el.buffer_size {
		el.events.PushBack(encodedEvent)
		el.cond.Signal()
	} else {
		log.WithFields(log.Fields{"eventListener": el.pool}).Error("events reaches the buffer_size, discard the events")
	}
}

func (el *EventListener) encodeEvent(event Event) []byte {
	body := []byte(event.GetBody())

	//header
	s := fmt.Sprintf("ver:%s server:%s serial:%d pool:%s poolserial:%d eventname:%s len:%d\n",
		EVENT_SYS_VERSION,
		el.server,
		event.GetSerial(),
		el.pool,
		eventPoolSerial.nextSerial(el.pool),
		event.GetType(),
		len(body))
	//write the header & body to buffer
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
	"TICK_5":                {"EVENT", "TICK"},
	"TICK_60":               {"EVENT", "TICK"},
	"TICK_3600":             {"EVENT", "TICK"},
	"PROCESS_GROUP_ADDED":   {"EVENT", "PROCESS_GROUP"},
	"PROCESS_GROUP_REMOVED": {"EVENT", "PROCESS_GROUP"}}
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

	//start a Tick timer
	go func() {
		lastTickSlice := make(map[string]int64)

		c := time.Tick(1 * time.Second)
		for now := range c {
			for tickType, period := range tickConfigs {
				time_slice := now.Unix() / period
				last_time_slice, ok := lastTickSlice[tickType]
				if !ok {
					lastTickSlice[tickType] = time_slice
				} else if last_time_slice != time_slice {
					lastTickSlice[tickType] = time_slice
					EmitEvent(NewTickEvent(tickType, now.Unix()))
				}
			}
		}
	}()
}

func nextEventSerial() uint64 {
	return atomic.AddUint64(&eventSerial, 1)
}

func NewEventListenerManager() *EventListenerManager {
	return &EventListenerManager{namedListeners: make(map[string]*EventListener),
		eventListeners: make(map[string]map[*EventListener]bool)}
}

func (em *EventListenerManager) registerEventListener(eventListenerName string,
	events []string,
	listener *EventListener) {

	em.namedListeners[eventListenerName] = listener
	all_events := make(map[string]bool)
	for _, event := range events {
		for k, values := range eventTypeDerives {
			if event == k { //if it is a final event
				all_events[k] = true
			} else { //if it is an abstract event, add all its derived events
				for _, val := range values {
					if val == event {
						all_events[k] = true
					}
				}
			}
		}
	}
	for event := range all_events {
		log.WithFields(log.Fields{"eventListener": eventListenerName, "event": event}).Info("register event listener")
		if _, ok := em.eventListeners[event]; !ok {
			em.eventListeners[event] = make(map[*EventListener]bool)
		}
		em.eventListeners[event][listener] = true
	}
}

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

func UnregisterEventListener(eventListenerName string) *EventListener {
	return eventListenerManager.unregisterEventListener(eventListenerName)
}

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

type RemoteCommunicationEvent struct {
	BaseEvent
	typ  string
	data string
}

func NewRemoteCommunicationEvent(typ string, data string) *RemoteCommunicationEvent {
	r := &RemoteCommunicationEvent{typ: typ, data: data}
	r.eventType = "REMOTE_COMMUNICATION"
	r.serial = nextEventSerial()
	return r
}

func (r *RemoteCommunicationEvent) GetBody() string {
	return fmt.Sprintf("type:%s\n%s", r.typ, r.data)
}

type ProcCommEvent struct {
	BaseEvent
	processName string
	groupName   string
	pid         int
	data        string
}

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

func (p *ProcCommEvent) GetBody() string {
	return fmt.Sprintf("processname:%s groupname:%s pid:%d\n%s", p.processName, p.groupName, p.pid, p.data)
}

func EmitEvent(event Event) {
	eventListenerManager.EmitEvent(event)
}

type TickEvent struct {
	BaseEvent
	when int64
}

func NewTickEvent(tickType string, when int64) *TickEvent {
	r := &TickEvent{when: when}
	r.eventType = tickType
	r.serial = nextEventSerial()
	return r
}

func (te *TickEvent) GetBody() string {
	return fmt.Sprintf("when:%d", te.when)
}

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
	end_pos := pec.findEndStr()
	if end_pos == -1 {
		return nil
	}
	data := pec.eventBuffer[pec.eventBeginPos+len(PROC_COMMON_BEGIN_STR) : end_pos]
	pec.eventBuffer = pec.eventBuffer[end_pos+len(PROC_COMMON_END_STR):]
	pec.eventBeginPos = -1
	return NewProcCommEvent(pec.stdType,
		pec.procName,
		pec.groupName,
		pec.pid,
		data)
}

func (pec *ProcCommEventCapture) findBeginStr() {
	if pec.eventBeginPos == -1 {
		pec.eventBeginPos = strings.Index(pec.eventBuffer, PROC_COMMON_BEGIN_STR)
		if pec.eventBeginPos == -1 {
			//remove some string
			n := len(pec.eventBuffer)
			if n > len(PROC_COMMON_BEGIN_STR) {
				pec.eventBuffer = pec.eventBuffer[n-len(PROC_COMMON_BEGIN_STR):]
			}
		}
	}
}

func (pec *ProcCommEventCapture) findEndStr() int {
	if pec.eventBeginPos == -1 {
		return -1
	}
	end_pos := strings.Index(pec.eventBuffer, PROC_COMMON_END_STR)
	if end_pos == -1 {
		if len(pec.eventBuffer) > pec.captureMaxBytes {
			log.WithFields(log.Fields{"program": pec.procName}).Warn("The capture buffer is overflow, discard the content")
			pec.eventBeginPos = -1
			pec.eventBuffer = ""
		}
	}
	return end_pos
}

type ProcessStateEvent struct {
	BaseEvent
	process_name string
	group_name   string
	from_state   string
	tries        int
	expected     int
	pid          int
}

func CreateProcessStartingEvent(process string,
	group string,
	from_state string,
	tries int) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      tries,
		expected:   -1,
		pid:        0}
	r.eventType = "PROCESS_STATE_STARTING"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessRunningEvent(process string,
	group string,
	from_state string,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      -1,
		expected:   -1,
		pid:        pid}
	r.eventType = "PROCESS_STATE_RUNNING"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessBackoffEvent(process string,
	group string,
	from_state string,
	tries int) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      tries,
		expected:   -1,
		pid:        0}
	r.eventType = "PROCESS_STATE_BACKOFF"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessStoppingEvent(process string,
	group string,
	from_state string,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      -1,
		expected:   -1,
		pid:        pid}
	r.eventType = "PROCESS_STATE_STOPPING"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessExitedEvent(process string,
	group string,
	from_state string,
	expected int,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      -1,
		expected:   expected,
		pid:        pid}
	r.eventType = "PROCESS_STATE_EXITED"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessStoppedEvent(process string,
	group string,
	from_state string,
	pid int) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      -1,
		expected:   -1,
		pid:        pid}
	r.eventType = "PROCESS_STATE_STOPPED"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessFatalEvent(process string,
	group string,
	from_state string) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      -1,
		expected:   -1,
		pid:        0}
	r.eventType = "PROCESS_STATE_FATAL"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessUnknownEvent(process string,
	group string,
	from_state string) *ProcessStateEvent {
	r := &ProcessStateEvent{process_name: process,
		group_name: group,
		from_state: from_state,
		tries:      -1,
		expected:   -1,
		pid:        0}
	r.eventType = "PROCESS_STATE_UNKNOWN"
	r.serial = nextEventSerial()
	return r
}

func (pse *ProcessStateEvent) GetBody() string {
	body := fmt.Sprintf("processname:%s groupname:%s from_state:%s", pse.process_name, pse.group_name, pse.from_state)
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

type SupervisorStateChangeEvent struct {
	BaseEvent
}

func (s *SupervisorStateChangeEvent) GetBody() string {
	return ""
}

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

type ProcessLogEvent struct {
	BaseEvent
	process_name string
	group_name   string
	pid          int
	data         string
}

func (pe *ProcessLogEvent) GetBody() string {
	return fmt.Sprintf("processname:%s groupname:%s pid:%d\n%s",
		pe.process_name,
		pe.group_name,
		pe.pid,
		pe.data)
}

func CreateProcessLogStdoutEvent(process_name string,
	group_name string,
	pid int,
	data string) *ProcessLogEvent {
	r := &ProcessLogEvent{process_name: process_name,
		group_name: group_name,
		pid:        pid,
		data:       data}
	r.eventType = "PROCESS_LOG_STDOUT"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessLogStderrEvent(process_name string,
	group_name string,
	pid int,
	data string) *ProcessLogEvent {
	r := &ProcessLogEvent{process_name: process_name,
		group_name: group_name,
		pid:        pid,
		data:       data}
	r.eventType = "PROCESS_LOG_STDERR"
	r.serial = nextEventSerial()
	return r
}

type ProcessGroupEvent struct {
	BaseEvent
	group_name string
}

func (pe *ProcessGroupEvent) GetBody() string {
	return fmt.Sprintf("groupname:%s", pe.group_name)
}

func CreateProcessGroupAddedEvent(group_name string) *ProcessGroupEvent {
	r := &ProcessGroupEvent{group_name: group_name}

	r.eventType = "PROCESS_GROUP_ADDED"
	r.serial = nextEventSerial()
	return r
}

func CreateProcessGroupRemovedEvent(group_name string) *ProcessGroupEvent {
	r := &ProcessGroupEvent{group_name: group_name}

	r.eventType = "PROCESS_GROUP_REMOVED"
	r.serial = nextEventSerial()
	return r
}
