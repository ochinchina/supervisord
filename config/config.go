package config

import (
	"path/filepath"

	"github.com/creasty/defaults"
	"github.com/stuartcarnie/gopm/model"
	"github.com/stuartcarnie/gopm/model/ini"
	"github.com/stuartcarnie/gopm/model/yaml"
)

// Config memory representations of supervisor configuration file
type Config struct {
	configFile string

	groups   map[string]*model.Group
	programs map[string]*model.Program

	HttpServer   *model.HTTPServer
	GrpcServer   *model.GrpcServer
	ProgramGroup *ProcessGroup
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

func (c *Config) CreateGroup(name string) *model.Group {
	obj := c.groups[name]
	if obj == nil {
		obj = new(model.Group)
		obj.Name = name
		c.groups[name] = obj
	}
	_ = defaults.Set(obj)
	return obj
}

func (c *Config) CreateProgram(name string) *model.Program {
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
	var (
		m   *model.Root
		err error
	)

	ext := filepath.Ext(c.configFile)
	if ext == ".yaml" || ext == ".yml" {
		var r yaml.Reader
		m, err = r.LoadPath(c.configFile)
	} else {
		var ii ini.Reader
		m, err = ii.LoadPath(c.configFile)
	}

	if err != nil {
		return nil, err
	}

	if err := Validate(m); err != nil {
		return nil, err
	}

	return ApplyUpdates(c, m)
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
