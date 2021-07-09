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

	// AlreadyStated already stated result code
	AlreadyStated = 60

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
func NewFault(code int, desc string) error {
	return &xmlrpc.Fault{Code: code, String: desc}
}
