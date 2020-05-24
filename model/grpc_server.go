package model

type GrpcServer struct {
	Address  string `yaml:"address" ini:"address"`
	Username string `yaml:"username" ini:"username"`
	Password string `yaml:"password" ini:"password"`
}
