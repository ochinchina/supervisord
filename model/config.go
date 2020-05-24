package model

type Root struct {
	InetHTTPServer *HTTPServer    `yaml:"http_server"`
	GrpcServer     *GrpcServer    `yaml:"grpc_server"`
	SupervisorCtl  *SupervisorCtl `yaml:"supervisor_ctl"`
	Programs       []*Program     `yaml:"programs"`
	Groups         []*Group       `yaml:"groups"`
}
