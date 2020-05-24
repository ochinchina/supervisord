package config

import (
	"github.com/stuartcarnie/gopm/model"
)

func ApplyUpdates(c *Config, m *model.Root) ([]string, error) {
	var u updater
	return u.update(c, m)
}

type updater struct{}

func (u *updater) update(c *Config, m *model.Root) ([]string, error) {
	c.ProgramGroup = NewProcessGroup()

	u.applyGroup(c, m)
	loadedPrograms := u.applyPrograms(c, m)
	u.applyInetHttpServer(c, m)
	u.applyGrpcServer(c, m)
	u.applySupervisorCtl(c, m)

	return loadedPrograms, nil
}

func (u *updater) applyGroup(c *Config, m *model.Root) {
	for _, g := range m.Groups {
		obj := c.CreateGroup(g.Name)
		*obj = *g
		for _, program := range obj.Programs {
			c.ProgramGroup.Add(obj.Name, program)
		}
	}
}

func (u *updater) applyPrograms(c *Config, m *model.Root) []string {
	loadedPrograms := make([]string, 0, len(m.Programs))
	for _, program := range m.Programs {
		programName := program.Name
		obj := c.CreateProgram(programName)
		*obj = *program
		group := c.ProgramGroup.GetGroup(programName, programName)
		obj.Group = group
		loadedPrograms = append(loadedPrograms, programName)
	}
	return loadedPrograms
}

func (u *updater) applyInetHttpServer(c *Config, m *model.Root) {
	if m.InetHTTPServer == nil {
		c.HTTPServer = nil
		return
	}
	if c.HTTPServer == nil {
		c.HTTPServer = new(model.HTTPServer)
	}
	*c.HTTPServer = *m.InetHTTPServer
}

func (u *updater) applyGrpcServer(c *Config, m *model.Root) {
	if m.GrpcServer == nil {
		c.GrpcServer = nil
		return
	}
	if c.GrpcServer == nil {
		c.GrpcServer = new(model.GrpcServer)
	}
	*c.GrpcServer = *m.GrpcServer
}

func (u *updater) applySupervisorCtl(c *Config, m *model.Root) {
	if m.SupervisorCtl == nil {
		c.SupervisorCtl = nil
		return
	}
	if c.SupervisorCtl == nil {
		c.SupervisorCtl = new(model.SupervisorCtl)
	}
	*c.SupervisorCtl = *m.SupervisorCtl
}
