package model

type Root struct {
	HttpServer *HTTPServer `yaml:"http_server"`
	GrpcServer *GrpcServer `yaml:"grpc_server"`
	Programs   []*Program  `yaml:"programs"`
	Groups     []*Group    `yaml:"groups"`
	FileSystem *FileSystem
}

type FileSystem struct {
	Root  string
	Files []*File
}

type File struct {
	Name    string
	Path    string
	Content string
}
