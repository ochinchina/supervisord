package model

type InetHTTPServer struct {
	Port     string `yaml:"port" ini:"port"`
	Username string `yaml:"username" ini:"username"`
	Password string `yaml:"password" ini:"password"`
}
