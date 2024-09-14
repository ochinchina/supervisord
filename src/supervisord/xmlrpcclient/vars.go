package xmlrpcclient

// https://github.com/Supervisor/supervisor/blob/ff7f18169bcc8091055f61279d0a63997d594148/supervisor/xmlrpc.py#L26-L44
var (
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

type ProcStatusInfo struct {
	Name        string `xml:"name" json:"name"`
	Group       string `xml:"group" json:"group"`
	Status      int    `xml:"status" json:"status"`
	Description string `xml:"description" json:"description"`
}

type AllProcStatusInfoReply struct {
	Value []ProcStatusInfo
}
