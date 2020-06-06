package model

type Group struct {
	Name     string   `yaml:"name"`
	Programs []string `yaml:"programs"`
}
