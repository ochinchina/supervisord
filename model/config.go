package model

type Root struct {
	HttpServer *HTTPServer `yaml:"http_server"`
	GrpcServer *GrpcServer `yaml:"grpc_server"`
	Programs   []*Program  `yaml:"programs"`
	Groups     []*Group    `yaml:"groups"`
}
