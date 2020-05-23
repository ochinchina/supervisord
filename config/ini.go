package config

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"supervisord/model"

	"github.com/creasty/defaults"
	"github.com/go-ini/ini"
)

type Ini struct {
	configFile string
}

func (ii *Ini) Load(c *Config) ([]string, error) {
	f, err := ini.Load(ii.configFile)
	if err != nil {
		return nil, err
	}

	f.BlockMode = false
	c.ProgramGroup = NewProcessGroup()

	includeFiles := ii.getIncludeFiles(f)
	for _, file := range includeFiles {
		err = f.Append(file)
		if err != nil {
			return nil, err
		}
	}
	return ii.parse(f, c), nil
}

func (ii *Ini) parse(cfg *ini.File, c *Config) []string {
	ii.parseGroup(cfg, c)
	loadedPrograms := ii.parsePrograms(cfg, c)
	ii.parseUnixHttpServer(cfg, c)
	ii.parseInetHttpServer(cfg, c)
	ii.parseSupervisorCtl(cfg, c)

	return loadedPrograms
}

func (ii *Ini) getIncludeFiles(cfg *ini.File) []string {
	result := make([]string, 0)
	if includeSection, err := cfg.GetSection("include"); err == nil {
		key, err := includeSection.GetKey("files")
		if err != nil {
			return nil
		}

		env := NewStringExpression("here", ii.GetConfigFileDir())
		files := key.Strings(",")
		for _, fRaw := range files {
			dir := ii.GetConfigFileDir()
			f, err := env.Eval(fRaw)
			if err != nil {
				continue
			}
			if filepath.IsAbs(f) {
				dir = filepath.Dir(f)
			}
			fileInfos, err := ioutil.ReadDir(dir)
			if err == nil {
				goPattern := toRegexp(filepath.Base(f))
				for _, fileInfo := range fileInfos {
					if matched, err := regexp.MatchString(goPattern, fileInfo.Name()); matched && err == nil {
						result = append(result, filepath.Join(dir, fileInfo.Name()))
					}
				}
			}
		}
	}
	return result
}

func (ii *Ini) parseGroup(cfg *ini.File, c *Config) {
	sections := cfg.Section("group")
	if sections == nil {
		return
	}

	for _, section := range sections.ChildSections() {
		entry := c.createEntry(section.Name(), ii.GetConfigFileDir())
		obj := new(model.Group)
		_ = defaults.Set(obj)
		ii.parseEntry(section, entry, obj)
		obj.Name = entry.Name[len("group."):]
		for _, program := range obj.Programs {
			c.ProgramGroup.Add(obj.Name, program)
		}
	}
}

func (ii *Ini) parseUnixHttpServer(cfg *ini.File, c *Config) {
	section, err := cfg.GetSection("unix_http_server")
	if err != nil {
		return
	}
	entry := c.createEntry("unix_http_server", ii.GetConfigFileDir())
	obj := new(model.UnixHTTPServer)
	_ = defaults.Set(obj)
	ii.parseEntry(section, entry, obj)
}

func (ii *Ini) parseInetHttpServer(cfg *ini.File, c *Config) {
	section, err := cfg.GetSection("inet_http_server")
	if err != nil {
		return
	}
	entry := c.createEntry("inet_http_server", ii.GetConfigFileDir())
	obj := new(model.InetHTTPServer)
	_ = defaults.Set(obj)
	ii.parseEntry(section, entry, obj)
}

func (ii *Ini) parseSupervisorCtl(cfg *ini.File, c *Config) {
	section, err := cfg.GetSection("supervisorctl")
	if err != nil {
		return
	}
	entry := c.createEntry("supervisorctl", ii.GetConfigFileDir())
	obj := new(model.SupervisorCtl)
	_ = defaults.Set(obj)
	ii.parseEntry(section, entry, obj)
}

// parse the sections starts with "program." prefix.
//
// Return all the parsed program names in the ini
func (ii *Ini) parsePrograms(cfg *ini.File, c *Config) []string {
	section, err := cfg.GetSection("program")
	if err != nil {
		return nil
	}

	sections := section.ChildSections()
	loadedPrograms := make([]string, 0, len(sections))
	for _, section := range sections {
		programName := section.Name()[len("program."):]
		entry := c.createEntry(programName, ii.GetConfigFileDir())
		obj := new(model.Program)
		_ = defaults.Set(obj)
		ii.parseEntry(section, entry, obj)
		obj.Name = programName
		obj.Command = stripEmpty(obj.Command)
		group := c.ProgramGroup.GetGroup(programName, programName)
		entry.Group = group
		loadedPrograms = append(loadedPrograms, programName)
	}
	return loadedPrograms
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

func (ii *Ini) parseEntry(section *ini.Section, c *Entry, v interface{}) {
	c.Name = section.Name()
	for _, key := range section.Keys() {
		c.keyValues[key.Name()] = strings.TrimSpace(key.MustString(""))
	}
	for _, key := range section.ParentKeys() {
		c.keyValues[key.Name()] = strings.TrimSpace(key.MustString(""))
	}
	if v != nil {
		_ = section.MapTo(v)
		c.Object = v
	}
}

// GetConfigFileDir get the directory of supervisor configuration file
func (ii *Ini) GetConfigFileDir() string {
	return filepath.Dir(ii.configFile)
}
