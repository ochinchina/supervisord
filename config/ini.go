package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"supervisord/model"

	"github.com/creasty/defaults"
	"github.com/go-ini/ini"
	"go.uber.org/zap"
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
	ii.parseProgramDefault(cfg, c)

	ii.parseGroup(cfg, c)
	loadedPrograms := ii.parseProgram(cfg, c)

	//parse non-group,non-program and non-eventlistener sections
	for _, section := range cfg.Sections() {
		switch section.Name() {
		case "program-default":
		case "unix_http_server":
		case "inet_http_server":
		case "supervisord":
		case "supervisorctl":
		default:
			// unrecognized section
		}
		if !strings.HasPrefix(section.Name(), "group:") && !strings.HasPrefix(section.Name(), "program:") && !strings.HasPrefix(section.Name(), "eventlistener:") {
			entry := c.createEntry(section.Name(), ii.GetConfigFileDir())
			c.entries[section.Name()] = entry
			ii.parseEntry(section, entry, nil)
		}
	}
	return loadedPrograms
}

func (ii *Ini) parseProgramDefault(cfg *ini.File, c *Config) {
	var p model.Program
	_ = defaults.Set(&p)

	section := cfg.Section("program-default")
	if section == nil {
		return
	}

	entry := c.createEntry("program-default", ii.GetConfigFileDir())
	ii.parseEntry(section, entry, nil)
	_ = section.MapTo(&p)
	entry.Object = &p
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
	//parse the group at first
	for _, section := range cfg.Sections() {
		if strings.HasPrefix(section.Name(), "group:") {
			entry := c.createEntry(section.Name(), ii.GetConfigFileDir())
			ii.parseEntry(section, entry, nil)
			groupName := entry.GetGroupName()
			programs := entry.GetPrograms()
			for _, program := range programs {
				c.ProgramGroup.Add(groupName, program)
			}
		}
	}
}

func (ii *Ini) isProgramOrEventListener(section *ini.Section) (bool, string) {
	//check if it is a program or event listener section
	isProgram := strings.HasPrefix(section.Name(), "program:")
	isEventListener := strings.HasPrefix(section.Name(), "eventlistener:")
	prefix := ""
	if isProgram {
		prefix = "program:"
	} else if isEventListener {
		prefix = "eventlistener:"
	}
	return isProgram || isEventListener, prefix
}

// parse the sections starts with "program:" prefix.
//
// Return all the parsed program names in the ini
func (ii *Ini) parseProgram(cfg *ini.File, c *Config) []string {
	loadedPrograms := make([]string, 0)
	for _, section := range cfg.Sections() {

		programOrEventListener, prefix := ii.isProgramOrEventListener(section)

		//if it is program or event listener
		if programOrEventListener {
			//get the number of processes
			programName := section.Name()[len(prefix):]
			numProcs := 1
			if numProcsKey, err := section.GetKey("numprocs"); err == nil {
				numProcs = numProcsKey.MustInt(1)
			}

			var procName string
			procNameKey, err := section.GetKey("process_name")
			if procNameKey != nil {
				procName = procNameKey.MustString("")
			}
			if numProcs > 1 {
				if err != nil || strings.Index(procName, "%(process_num)") == -1 {
					zap.L().Error("no process_num in process name", zap.Int("numprocs", numProcs), zap.String("process_name", procName))
				}
			}
			originalProcName := programName
			if err == nil {
				originalProcName = procName
			}

			for i := 1; i <= numProcs; i++ {
				envs := NewStringExpression("program_name", programName,
					"process_num", fmt.Sprintf("%d", i),
					"group_name", c.ProgramGroup.GetGroup(programName, programName),
					"here", ii.GetConfigFileDir())
				var cmd string
				if cmdKey, err := section.GetKey("command"); err == nil {
					cmd = cmdKey.MustString("")
				}
				cmd, err := envs.Eval(cmd)
				if err != nil {
					zap.L().Error("get envs failed", zap.Error(err), zap.String("program", programName))
					continue
				}
				section.DeleteKey("command")
				_, _ = section.NewKey("command", cmd)

				procName, err := envs.Eval(originalProcName)
				if err != nil {
					zap.L().Error("get envs failed", zap.Error(err), zap.String("program", "programName"))
					continue
				}

				section.DeleteKey("process_name")
				_, _ = section.NewKey("process_name", procName)
				section.DeleteKey("process_num")
				_, _ = section.NewKey("process_num", fmt.Sprintf("%d", i))
				entry := c.createEntry(procName, ii.GetConfigFileDir())
				ii.parseEntry(section, entry, nil)
				entry.Name = prefix + procName
				group := c.ProgramGroup.GetGroup(programName, programName)
				entry.Group = group
				loadedPrograms = append(loadedPrograms, procName)
			}
		}
	}
	return loadedPrograms
}

func (ii *Ini) parseEntry(section *ini.Section, c *Entry, v interface{}) {
	c.Name = section.Name()
	for _, key := range section.Keys() {
		c.keyValues[key.Name()] = strings.TrimSpace(key.MustString(""))
	}
	if v != nil {
		_ = section.MapTo(v)
	}
}

// GetConfigFileDir get the directory of supervisor configuration file
func (ii *Ini) GetConfigFileDir() string {
	return filepath.Dir(ii.configFile)
}
