package main

import (
	"fmt"
	"github.com/go-ini/ini"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	log "github.com/Sirupsen/logrus"
)

type ConfigEntry struct {
	ConfigDir string
	Group	string
	Name      string
	keyValues map[string]string
	dict	map[string]string
}

func (c *ConfigEntry) IsProgram() bool {
	return strings.HasPrefix(c.Name, "program:")
}

func (c *ConfigEntry) GetProgramName() string {
	if strings.HasPrefix(c.Name, "program:") {
		return c.Name[len("program:"):]
	}
	return ""
}

func (c *ConfigEntry) IsGroup() bool {
	return strings.HasPrefix( c.Name, "group:" )
}

// get the group name if this entry is group
func (c *ConfigEntry) GetGroupName() string {
	if strings.HasPrefix( c.Name, "group:" ) {
		return c.Name[len("group:"):]
	} else {
		return ""
	}
}

// get the programs from the group
func (c *ConfigEntry) GetPrograms() []string {
	if c.IsGroup() {
		return c.GetStringArray( "programs", "," )
	}
	return make( []string, 0 )
}


func (c* ConfigEntry) setGroup( group string ) {
	c.Group = group
	c.dict["group_name"] = group
}


type Config struct {
	configFile string
	//mapping between the section name and the configure
	entries map[string]*ConfigEntry

	programGroup *ProcessGroup
}

func NewConfigEntry( configDir string ) *ConfigEntry {
	return &ConfigEntry{configDir, "", "", make(map[string]string), make(map[string]string) }
}

func NewConfig( configFile string ) *Config {
	return &Config{configFile , make(map[string]*ConfigEntry), NewProcessGroup() }
}

//create a new entry or return the already-exist entry
func (c *Config) createEntry( name string, configDir string ) *ConfigEntry {
	entry, ok := c.entries[name]

	if !ok {
		entry = NewConfigEntry( configDir )
		c.entries[name] = entry 
	}
	return entry
}

func (c *Config)Load() error  {
	cfg, err := ini.Load(c.configFile )

	if err != nil {
		return err
	}
	includeFiles := c.getIncludeFiles( cfg )
	for _, f := range includeFiles {
		err = cfg.Append( f )
		if err != nil {
			return err
		}
	}
	c.parse( cfg )
	return nil
}

func (c *Config) getIncludeFiles( cfg *ini.File) []string {
	result := make([]string,0)
	if includeSection, err := cfg.GetSection("include"); err == nil {
                key, err := includeSection.GetKey("files")
		if err == nil && key != nil {
			env := NewStringExpression("here", c.GetConfigFileDir() )
			files := strings.Fields( key.Value() )
			for _, f := range files {
				dir := c.GetConfigFileDir()
				f, err := env.Eval( f )
				if err != nil {
					continue
				}
				if filepath.IsAbs( f ) {
                                        dir = filepath.Dir( f )
                                }
                                fileInfos, err := ioutil.ReadDir( dir )
                                if err == nil {
                                        goPattern := toRegexp( filepath.Base( f ) )
                                        for _, fileInfo := range fileInfos {
                                                if matched, err := regexp.MatchString( goPattern, fileInfo.Name() ); matched && err == nil {
							result = append( result, filepath.Join( dir, fileInfo.Name() ) )
                                                }
                                        }
                                }

			}
		}
        }
	return result

}


func (c *Config) parse( cfg *ini.File) {
	c.parseGroup( cfg)
        c.parseProgram( cfg )
        for _, section := range cfg.Sections() {
                if !strings.HasPrefix( section.Name(), "group:" ) && !strings.HasPrefix( section.Name(), "program:" ) {
                        entry := c.createEntry( section.Name(), c.GetConfigFileDir() )
                        c.entries[section.Name()] = entry
                        entry.parse(section)
                }
        }
}

func (c *Config) GetConfigFileDir() string {
	return filepath.Dir( c.configFile )
}

//convert supervisor file pattern to the go regrexp
func toRegexp( pattern string ) string {
	tmp := strings.Split( pattern, "." )
	for i, t := range tmp {
		s := strings.Replace( t, "*", ".*", -1 )
		tmp[i] = strings.Replace( s, "?", ".", -1 ) 
	}
	return strings.Join(tmp, "\\." )
}

//get the unix_http_server section
func (c *Config) GetUnixHttpServer() (*ConfigEntry, bool) {
	entry, ok := c.entries["unix_http_server"]

	return entry, ok
}

//get the supervisord section
func (c *Config) GetSupervisord()( *ConfigEntry, bool ) {
	entry, ok := c.entries["supervisord"]
	return entry, ok
}

// Get the inet_http_server configuration section
func (c *Config) GetInetHttpServer()( *ConfigEntry, bool ) {
	entry, ok := c.entries["inet_http_server"]
	return entry, ok
}

func (c* Config) GetGroups() []*ConfigEntry {
	result := make( []*ConfigEntry, 0 )
	for key, value := range c.entries {
		if strings.HasPrefix( key, "group:" ) {
			result = append( result, value )
		}
	}
	return result
}

func (c *Config) GetPrograms() []*ConfigEntry {
	result := make([]*ConfigEntry, 0)
	for _, entry := range c.entries {
		if entry.IsProgram() {
			result = append(result, entry)
		}
	}
	return result
}

func (c *Config)GetProgramNames()[]string {
	result := make([]string,0)
	for _, entry := range c.entries {
		if entry.IsProgram() {
			result = append( result, entry.GetProgramName() )
		}
	}
	return result
}

//return the proram configure entry or nil
func (c* Config) GetProgram( name string) *ConfigEntry {
	for _, entry := range c.entries {
		if entry.IsProgram() && entry.GetProgramName() == name {
			return entry
		}
	}
	return nil
}
// get value of key as bool
func (c *ConfigEntry) GetBool(key string, defValue bool) bool {
	value, ok := c.keyValues[key]

	if ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defValue
}

func toInt(s string, factor int, defValue int) int {
	i, err := strconv.Atoi(s)
	if err == nil {
		return i * factor
	} else {
		return defValue
	}
}

// get the value of the key as int
func (c *ConfigEntry) GetInt(key string, defValue int) int {
	value, ok := c.keyValues[key]

	if ok {
		return toInt(value, 1, defValue)
	} else {
		return defValue
	}
}

// get the value of key as environment setting. An enviroment string example:
//  environment = A="env 1",B="this is a test"
func (c *ConfigEntry) GetEnv(key string) []string {
	value, ok := c.keyValues[key]
	env := make([]string, 0)

	if ok {
		start := 0
		n := len(value)
		i := 0
		for {
			for i = start; i < n && value[i] != '='; {
				i += 1
			}
			key := value[start:i]
			start = i + 1
			if value[start] == '"' {
				for i = start + 1; i < n && value[i] != '"'; {
					i += 1
				}
				if i < n {
					env = append(env, fmt.Sprintf("%s=\"%s\"", key, value[start+1:i]))
				}
				if i+1 < n && value[i+1] == ',' {
					start = i + 2
				} else {
					break
				}
			} else {
				for i = start; i < n && value[i] != ','; {
					i += 1
				}
				if i < n {
					env = append(env, fmt.Sprintf("%s=\"%s\"", key, value[start:i]))
					start = i + 1
				} else {
					env = append(env, fmt.Sprintf("%s=\"%s\"", key, value[start:]))
					break
				}
			}
		}
	}

	result := make([]string,0)
	for i := 0; i < len( env ); i++ {
		tmp, err := NewStringExpression( "program_name", c.GetProgramName(),
                             "process_num", c.GetString("process_num", "0"),
                             "group_name", c.GetGroupName(),
                             "here", c.ConfigDir ).Eval( env[i] )
		if err == nil {
			result = append( result, tmp )
		}
	}
	return result
}

//get the value of key as string
func (c *ConfigEntry) GetString(key string, defValue string) string {
	s, ok := c.keyValues[key]

	if ok {
		return s
	} else {
		return defValue
	}
}

func (c *ConfigEntry) GetStringArray( key string, sep string ) []string {
	s, ok := c.keyValues[key]

	if ok {
		return strings.Split( s, sep )
	} else {
		return make([]string, 0 )
	}
}


// get the value of key as the bytes setting.
//
//	logSize=1MB
//	logSize=1GB
//	logSize=1KB
//	logSize=1024
//
func (c *ConfigEntry) GetBytes(key string, defValue int) int {
	v, ok := c.keyValues[key]

	if ok {
		if len( v ) > 2 {
			lastTwoBytes := v[len(v)-2:]
			if lastTwoBytes == "MB" {
				return toInt(v[:len(v)-2], 1024*1024, defValue)
			} else if lastTwoBytes == "GB"{
				return toInt(v[:len(v)-2], 1024*1024*1024, defValue)
			} else if lastTwoBytes == "KB" {
				return toInt(v[:len(v)-2], 1024, defValue)
			}
		}
		return toInt(v, 1, defValue)
	} else {
		return defValue
	}
}

func (c *ConfigEntry) parse(section *ini.Section) {
	c.Name = section.Name()
	for _, key := range section.Keys() {
		c.keyValues[key.Name()] = key.Value()
	}
}

func (c *Config) parseGroup( cfg *ini.File ) {

        //parse the group at first
        for _, section := range cfg.Sections() {
                if strings.HasPrefix( section.Name(), "group:" ) {
                        entry := c.createEntry( section.Name(), c.GetConfigFileDir() )
                        entry.parse( section )
                        groupName := entry.GetGroupName()
                        programs := entry.GetPrograms()
                        for _, program := range( programs ) {
                                c.programGroup.Add( groupName, program )
                        }
                }
        }
}

func (c *Config) parseProgram( cfg *ini.File ) {
	for _, section := range cfg.Sections() {
                if strings.HasPrefix( section.Name(), "program:" ) {
                        numProcs, err := section.Key( "numprocs" ).Int()
                        programName :=  section.Name()[len("program:"):]
                        if err != nil {
                                numProcs = 1
                        }
			if numProcs > 1 {
				procNameKey, err := section.GetKey( "process_name" )
				if err != nil || strings.Index( procNameKey.Value(), "%(process_num)" ) == -1 {
					log.WithFields( log.Fields{
						"numprocs": numProcs,
						"process_name": procNameKey.Value(),
					}).Error( "no process_num in process name" )
				}
			}
                        for i := 1; i <= numProcs; i += 1 {
                                envs := NewStringExpression( "program_name", programName, 
							"process_num", fmt.Sprintf( "%d", i ), 
							"group_name", c.programGroup.GetGroup(programName, programName ),
							"here", c.GetConfigFileDir() )
				cmd, err := envs.Eval(section.Key( "command" ).Value() )
				if err != nil {
					continue
				}
                                section.NewKey( "command", cmd )
                                var procName string = ""
                                procNameKey, err := section.GetKey( "process_name" )

                                if err != nil {
                                        procName = programName
                                } else {
                                        procName, err = envs.Eval( procNameKey.Value() )
					if err != nil {
						continue
					}
                                }
                                section.NewKey( "process_name", procName )
                                section.NewKey( "numprocs_start", fmt.Sprintf("%d", (i - 1 ) ) )
                                section.NewKey( "process_num", fmt.Sprintf( "%d", i ) )
                                entry := c.createEntry(procName, c.GetConfigFileDir() )
                                entry.parse(section)
                                entry.Name = "program:" + procName
                                group := c.programGroup.GetGroup(programName, programName )
                                entry.Group = group
                        }
                }
	}

}

