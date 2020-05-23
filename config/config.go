package config

import (
	"strings"
	"supervisord/model"

	"github.com/creasty/defaults"
)

// Config memory representations of supervisor configuration file
type Config struct {
	configFile string

	groups   map[string]*model.Group
	programs map[string]*model.Program

	InetHTTPServer *model.InetHTTPServer
	UnixHTTPServer *model.UnixHTTPServer
	SupervisorCtl  *model.SupervisorCtl
	ProgramGroup   *ProcessGroup
}

// NewConfig create Config object
func NewConfig(configFile string) *Config {
	return &Config{
		configFile:   configFile,
		groups:       make(map[string]*model.Group),
		programs:     make(map[string]*model.Program),
		ProgramGroup: NewProcessGroup(),
	}
}

func (c *Config) createGroup(name string) *model.Group {
	obj := c.groups[name]
	if obj == nil {
		obj = new(model.Group)
		obj.Name = name
		c.groups[name] = obj
	}
	_ = defaults.Set(obj)
	return obj
}

func (c *Config) createProgram(name string) *model.Program {
	obj := c.programs[name]
	if obj == nil {
		obj = new(model.Program)
		obj.Name = name
		c.programs[name] = obj
	}
	_ = defaults.Set(obj)
	return obj
}

//
// Load load the configuration and return the loaded programs
func (c *Config) Load() ([]string, error) {
	ii := Ini{c.configFile}
	return ii.Load(c)
}

// convert supervisor file pattern to the go regrexp
func toRegexp(pattern string) string {
	tmp := strings.Split(pattern, ".")
	for i, t := range tmp {
		s := strings.Replace(t, "*", ".*", -1)
		tmp[i] = strings.Replace(s, "?", ".", -1)
	}
	return strings.Join(tmp, "\\.")
}

func (c *Config) Programs() model.Programs {
	res := make(model.Programs, 0)
	for _, p := range c.programs {
		res = append(res, p)
	}
	return res.Sorted()
}

// ProgramNames returns the names of all programs
func (c *Config) ProgramNames() []string {
	return c.Programs().Names()
}

// GetProgram return the proram configure entry or nil
func (c *Config) GetProgram(name string) *model.Program {
	return c.programs[name]
}

// RemoveProgram remove a program entry by its name
func (c *Config) RemoveProgram(programName string) {
	delete(c.programs, programName)
	c.ProgramGroup.Remove(programName)
}
