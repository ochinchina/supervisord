package ini

import (
	"io"

	"github.com/creasty/defaults"
	"github.com/stuartcarnie/gopm/model"
	"gopkg.in/ini.v1"
)

type Reader struct{}

func (ii *Reader) LoadReader(r io.Reader) (*model.Root, error) {
	f, err := ini.Load(r)
	if err != nil {
		return nil, err
	}

	f.BlockMode = false
	return ii.loadFile(f)
}

func (ii *Reader) LoadPath(path string) (*model.Root, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	f.BlockMode = false
	return ii.loadFile(f)
}

func (ii *Reader) loadFile(f *ini.File) (*model.Root, error) {
	c := new(model.Root)
	err := ii.parse(f, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (ii *Reader) parse(f *ini.File, c *model.Root) error {
	ii.parseGroup(f, c)
	ii.parsePrograms(f, c)
	ii.parseHttpServer(f, c)
	ii.parseGrpcServer(f, c)
	ii.parseSupervisorCtl(f, c)
	return nil
}

func (ii *Reader) parseGroup(f *ini.File, c *model.Root) {
	for _, section := range f.ChildSections("group") {
		groupName := section.Name()[len("group."):]
		obj := new(model.Group)
		_ = defaults.Set(obj)
		_ = section.MapTo(obj)
		obj.Name = groupName
		c.Groups = append(c.Groups, obj)
	}
}

// parse the sections starts with "program." prefix.
//
// Return all the parsed program names in the ini
func (ii *Reader) parsePrograms(cfg *ini.File, c *model.Root) {
	sections := cfg.ChildSections("program")
	for _, section := range sections {
		programName := section.Name()[len("program."):]
		obj := new(model.Program)
		_ = defaults.Set(obj)
		_ = section.MapTo(obj)
		obj.Name = programName
		obj.Environment = stripEmpty(obj.Environment)
		c.Programs = append(c.Programs, obj)
	}
}

func (ii *Reader) parseHttpServer(cfg *ini.File, c *model.Root) {
	section, err := cfg.GetSection("http_server")
	if err != nil {
		return
	}
	obj := c.InetHTTPServer
	if obj == nil {
		obj = new(model.HTTPServer)
	}
	_ = defaults.Set(obj)
	_ = section.MapTo(obj)
	c.InetHTTPServer = obj
}

func (ii *Reader) parseGrpcServer(cfg *ini.File, c *model.Root) {
	section, err := cfg.GetSection("grpc_server")
	if err != nil {
		return
	}
	obj := c.GrpcServer
	if obj == nil {
		obj = new(model.GrpcServer)
	}
	_ = defaults.Set(obj)
	_ = section.MapTo(obj)
	c.GrpcServer = obj
}

func (ii *Reader) parseSupervisorCtl(cfg *ini.File, c *model.Root) {
	section, err := cfg.GetSection("supervisorctl")
	if err != nil {
		return
	}
	obj := c.SupervisorCtl
	if obj == nil {
		obj = new(model.SupervisorCtl)
	}
	_ = defaults.Set(obj)
	_ = section.MapTo(obj)
	c.SupervisorCtl = obj
}

func stripEmpty(strings []string) []string {
	emptyCount := 0
	for _, s := range strings {
		if len(s) == 0 {
			emptyCount++
		}
	}
	if emptyCount == 0 {
		return strings
	}

	res := make([]string, 0, len(strings)-emptyCount)
	for _, s := range strings {
		if len(s) > 0 {
			res = append(res, s)
		}
	}
	return res
}
