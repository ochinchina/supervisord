package faults

import (
	xmlrpc "github.com/ochinchina/gorilla-xmlrpc/xml"
)

const (
	// UnknownMethod unknown xml rpc method
	UnknownMethod = 1
	// IncorrectParameters  incorrect parameters result code
	IncorrectParameters = 2

	// BadArguments Bad argument result code for xml rpc
	BadArguments = 3

	// SignatureUnsupported signature unsupported result code for xml rpc
	SignatureUnsupported = 4

	// ShutdownState shutdown state result code
	ShutdownState = 6

	// BadName bad name result code
	BadName = 10

	// BadSignal bad signal result code
	BadSignal = 11
	// NoFile no such file result code
	NoFile = 20

	// NotExecutable not executable result code
	NotExecutable = 21

	// Failed failed result code
	Failed = 30

	// AbnormalTermination abnormal termination result code
	AbnormalTermination = 40

	// SpawnError spawn error result code
	SpawnError = 50

	// AlreadyStarted already started result code
	AlreadyStarted = 60

	// NotRunning not running result code
	NotRunning = 70

	// Success success result code
	Success = 80

	// AlreadyAdded already added result code
	AlreadyAdded = 90

	// StillRunning still running result code
	StillRunning = 91

	// CantReRead can't re-read result code
	CantReRead = 92
)

// NewFault creates Fault object as xml rpc result
func NewFault(code int, desc string) xmlrpc.Fault {
	return xmlrpc.Fault{Code: code, String: desc}
}

func FaultCodeToString(faultCode int) string {
	switch faultCode {
	case UnknownMethod:
		return "Unknown method"
	case IncorrectParameters:
		return "Incorrect parameters"
	case BadArguments:
		return "Bad arguments"
	case SignatureUnsupported:
		return "Signature unsupported"
	case ShutdownState:
		return "Shutdown state"
	case BadName:
		return "Bad name"
	case BadSignal:
		return "Bad signal"
	case NoFile:
		return "No such file"
	case NotExecutable:
		return "Not executable"
	case Failed:
		return "Failed"
	case AbnormalTermination:
		return "Abnormal termination"
	case SpawnError:
		return "Spawn error"
	case AlreadyStarted:
		return "Already stated"
	case NotRunning:
		return "Not running"
	case Success:
		return "Success"
	case AlreadyAdded:
		return "Already added"
	case StillRunning:
		return "Still running"
	case CantReRead:
		return "Can't re-read"
	default:
		return "Unknown fault"
	}
}

func GetANSIColorByFaultCode(faultCode int) string {
	switch faultCode {
	case UnknownMethod:
		return "\033[31m"
	case IncorrectParameters:
		return "\033[31m"
	case BadArguments:
		return "\033[31m"
	case SignatureUnsupported:
		return "\033[31m"
	case ShutdownState:
		return "\033[33m"
	case BadName:
		return "\033[31m"
	case BadSignal:
		return "\033[31m"
	case NoFile:
		return "\033[31m"
	case NotExecutable:
		return "\033[31m"
	case Failed:
		return "\033[31m"
	case AbnormalTermination:
		return "\033[31m"
	case SpawnError:
		return "\033[31m"
	case AlreadyStarted:
		return "\033[33m"
	case NotRunning:
		return "\033[33m"
	case Success:
		return "\033[32m"
	case AlreadyAdded:
		return "\033[33m"
	case StillRunning:
		return "\033[33m"
	case CantReRead:
		return "\033[31m"
	default:
		return "\033[0m"
	}
}
