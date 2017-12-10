package faults

import (
	xmlrpc "github.com/ochinchina/gorilla-xmlrpc/xml"
)

const (
	UNKNOWN_METHOD        = 1
	INCORRECT_PARAMETERS  = 2
	BAD_ARGUMENTS         = 3
	SIGNATURE_UNSUPPORTED = 4
	SHUTDOWN_STATE        = 6
	BAD_NAME              = 10
	BAD_SIGNAL            = 11
	NO_FILE               = 20
	NOT_EXECUTABLE        = 21
	FAILED                = 30
	ABNORMAL_TERMINATION  = 40
	SPAWN_ERROR           = 50
	ALREADY_STARTED       = 60
	NOT_RUNNING           = 70
	SUCCESS               = 80
	ALREADY_ADDED         = 90
	STILL_RUNNING         = 91
	CANT_REREAD           = 92
)

func NewFault(code int, desc string) error {
	return &xmlrpc.Fault{Code: code, String: desc}
}
