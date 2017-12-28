// +build windows

package process

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/alexbrainman/ps"
	"github.com/alexbrainman/ps/winapi"
)

const (
	JobObjectBasicLimitInformation    = 2
	JobObjectExtendedLimitInformation = 9
)

const (
	JOB_OBJECT_LIMIT_ACTIVE_PROCESS             = 0x00000008
	JOB_OBJECT_LIMIT_AFFINITY                   = 0x00000010
	JOB_OBJECT_LIMIT_BREAKAWAY_OK               = 0x00000800
	JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION = 0x00000400
	JOB_OBJECT_LIMIT_JOB_MEMORY                 = 0x00000200
	JOB_OBJECT_LIMIT_JOB_TIME                   = 0x00000004
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE          = 0x00002000
	JOB_OBJECT_LIMIT_PRESERVE_JOB_TIME          = 0x00000040
	JOB_OBJECT_LIMIT_PRIORITY_CLASS             = 0x00000020
	JOB_OBJECT_LIMIT_PROCESS_MEMORY             = 0x00000100
	JOB_OBJECT_LIMIT_PROCESS_TIME               = 0x00000002
	JOB_OBJECT_LIMIT_SCHEDULING_CLASS           = 0x00000080
	JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK        = 0x00001000
	JOB_OBJECT_LIMIT_SUBSET_AFFINITY            = 0x00004000
	JOB_OBJECT_LIMIT_WORKINGSET                 = 0x00000001
)

// Process Security and Access Rights
const (
	PROCESS_TERMINATE uint32 = 1
	PROCESS_SET_QUOTA uint32 = 0x0100
)

var (
	globalJobObject *ps.JobObject
)

func jobFinalizer(jo *ps.JobObject) error {
	return jo.Terminate(0)
}

func init() {
	var err error
	globalJobObject, err = ps.CreateJobObject("")
	if err != nil {
		panic(err)
	}
	runtime.SetFinalizer(globalJobObject, jobFinalizer)
	err = setInfoJobObject(globalJobObject.Handle)
	if err != nil {
		panic(err)
	}
}

func setInfoJobObject(jobHandle syscall.Handle) error {
	var info winapi.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_BREAKAWAY_OK |
		JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK |
		JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION |
		JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	return winapi.SetInformationJobObject(jobHandle, JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
}

func (p *Process) postStart() error {
	proc := p.cmd.Process
	// OpenProcess with specified access right
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681949(v=vs.85).aspx
	// The handle must have the PROCESS_SET_QUOTA and PROCESS_TERMINATE access rights.
	da := PROCESS_SET_QUOTA | PROCESS_TERMINATE
	h, err := syscall.OpenProcess(da, false, uint32(proc.Pid))
	if err != nil {
		return err
	}
	return globalJobObject.AddProcess(h)
}
