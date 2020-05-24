package model

type SupervisorCtl struct {
	ServerURL string `yaml:"server_url" ini:"serverurl"`
	Username  string `yaml:"username" ini:"username"`
	Password  string `yaml:"password" ini:"password"`
}
