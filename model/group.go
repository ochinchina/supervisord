package model

type Group struct {
	Name     string   `yaml:"name" ini:"-"`
	Programs []string `yaml:"programs" ini:"programs" delim:","`
}
