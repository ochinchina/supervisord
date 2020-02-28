// +build !windows,!nacl,!plan9

package logger

import (
	"errors"
	"fmt"
	"log/syslog"
	"strconv"
	"strings"
)

// NewSysLogger create a local syslog
func NewSysLogger(name string, logEventEmitter LogEventEmitter) *SysLogger {
	writer, err := syslog.New(syslog.LOG_DEBUG, name)
	logger := &SysLogger{logEventEmitter: logEventEmitter}
	if err == nil {
		logger.logWriter = writer
	}
	return logger
}

// BackendSysLogWriter a syslog writer to write the log to syslog in background
type BackendSysLogWriter struct {
	network    string
	raddr      string
	priority   syslog.Priority
	tag        string
	logChannel chan []byte
}

// NewBackendSysLogWriter create a backgroud running syslog writer
func NewBackendSysLogWriter(network, raddr string, priority syslog.Priority, tag string) *BackendSysLogWriter {
	bs := &BackendSysLogWriter{network: network, raddr: raddr, priority: priority, tag: tag, logChannel: make(chan []byte)}
	bs.start()
	return bs
}

func (bs *BackendSysLogWriter) start() {
	go func() {
		var writer *syslog.Writer = nil
		for {
			b, ok := <-bs.logChannel
			// if channel is closed
			if !ok {
				if writer != nil {
					writer.Close()
				}
				break
			}
			//if not connect to syslog, try to connect to it
			if writer == nil {
				writer, _ = syslog.Dial(bs.network, bs.raddr, bs.priority, bs.tag)
			}
			if writer != nil {
				writer.Write(b)
			}

		}
	}()
}

// Write write data to the backend syslog writer
func (bs *BackendSysLogWriter) Write(b []byte) (int, error) {
	bs.logChannel <- b
	return len(b), nil
}

// Close close the backgroup write channel
func (bs *BackendSysLogWriter) Close() error {
	close(bs.logChannel)
	return nil
}

// parse the configuration for syslog
// the configure should be in following format:
// [protocol:]host[:port]
//
// - protocol, should be tcp or udp
// - port, if missing, for tcp it should be 6514 and for udp it should be 514
//
func parseSysLogConfig(config string) (protocol string, host string, port int, err error) {
	fields := strings.Split(config, ":")
	host = ""
	protocol = ""
	port = 0
	err = nil
	switch len(fields) {
	case 1:
		host = fields[0]
		port = 514
		protocol = "udp"
	case 2:
		switch fields[0] {
		case "tcp":
			host = fields[1]
			protocol = "tcp"
			port = 6514
		case "udp":
			host = fields[1]
			protocol = "udp"
			port = 514
		default:
			protocol = "udp"
			host = fields[0]
			port, err = strconv.Atoi(fields[1])
			if err != nil {
				return
			}
		}
	case 3:
		protocol = fields[0]
		host = fields[1]
		port, err = strconv.Atoi(fields[2])
		if err != nil {
			return
		}
	default:
		err = errors.New("invalid format")
	}
	return

}

// NewRemoteSysLogger create a network syslog
func NewRemoteSysLogger(name string, config string, logEventEmitter LogEventEmitter) *SysLogger {
	if len(config) <= 0 {
		return NewSysLogger(name, logEventEmitter)
	}

	protocol, host, port, err := parseSysLogConfig(config)
	if err != nil {
		return NewSysLogger(name, logEventEmitter)
	}
	writer, err := syslog.Dial(protocol, fmt.Sprintf("%s:%d", host, port), syslog.LOG_LOCAL7|syslog.LOG_DEBUG, name)
	logger := &SysLogger{logEventEmitter: logEventEmitter}
	if writer != nil && err == nil {
		logger.logWriter = writer
	} else {
		logger.logWriter = NewBackendSysLogWriter(protocol, fmt.Sprintf("%s:%d", host, port), syslog.LOG_LOCAL7|syslog.LOG_DEBUG, name)
	}
	return logger

}
