package ini

import (
	"io"

	"github.com/creasty/defaults"
	"github.com/stuartcarnie/gopm/model"
	"gopkg.in/ini.v1"
)

type Reader struct{}

func (r *Reader) LoadReader(reader io.Reader) (*model.Root, error) {
	f, err := ini.Load(reader)
	if err != nil {
		return nil, err
	}

	f.BlockMode = false
	return r.loadFile(f)
}

func (r *Reader) LoadPath(path string) (*model.Root, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	f.BlockMode = false
	return r.loadFile(f)
}

func (r *Reader) loadFile(f *ini.File) (*model.Root, error) {
	c := new(model.Root)
	err := r.parse(f, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Reader) parse(f *ini.File, c *model.Root) error {
	r.parseGroup(f, c)
	r.parsePrograms(f, c)
	r.parseHttpServer(f, c)
	r.parseGrpcServer(f, c)
	return nil
}

func (r *Reader) parseGroup(f *ini.File, c *model.Root) {
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
func (r *Reader) parsePrograms(cfg *ini.File, c *model.Root) {
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

func (r *Reader) parseHttpServer(cfg *ini.File, c *model.Root) {
	section, err := cfg.GetSection("http_server")
	if err != nil {
		return
	}
	obj := c.HttpServer
	if obj == nil {
		obj = new(model.HTTPServer)
	}
	_ = defaults.Set(obj)
	_ = section.MapTo(obj)
	c.HttpServer = obj
}

func (r *Reader) parseGrpcServer(cfg *ini.File, c *model.Root) {
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
