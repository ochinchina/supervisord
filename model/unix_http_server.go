package model

type UnixHTTPServer struct {
	File     string `yaml:"file" ini:"file" default:"/tmp/supervisord.sock"`
	Username string `yaml:"username" ini:"username"`
	Password string `yaml:"password" ini:"password"`
}
