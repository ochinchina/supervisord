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
	u.applyHttpServer(c, m)
	u.applyGrpcServer(c, m)

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

func (u *updater) applyHttpServer(c *Config, m *model.Root) {
	if m.HttpServer == nil {
		c.HttpServer = nil
		return
	}
	if c.HttpServer == nil {
		c.HttpServer = new(model.HTTPServer)
	}
	*c.HttpServer = *m.HttpServer
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
