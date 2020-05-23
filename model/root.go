package model

type Root struct {
	SupervisorCtl  *SupervisorCtl
	InetHTTPServer *InetHTTPServer
	UnixHTTPServer *UnixHTTPServer
	Programs       Programs
}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) GetPrograms() []*Program {
	return r.Programs.GetPrograms()
}

func (r *Root) GetProgramNames() []string {
	return r.Programs.GetProgramNames()
}
