package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// Entry standards for a configuration section in supervisor configuration file
type Entry struct {
	ConfigDir string
	Group     string
	Name      string
	keyValues map[string]string
	Object    interface{}
}

// IsProgram return true if this is a program section
func (c *Entry) IsProgram() bool {
	return strings.HasPrefix(c.Name, "program.")
}

// GetProgramName get the program name
func (c *Entry) GetProgramName() string {
	if strings.HasPrefix(c.Name, "program.") {
		return c.Name[len("program."):]
	}
	return ""
}

// IsGroup return true if it is group section
func (c *Entry) IsGroup() bool {
	return strings.HasPrefix(c.Name, "group.")
}

// GetGroupName get the group name if this entry is group
func (c *Entry) GetGroupName() string {
	if strings.HasPrefix(c.Name, "group.") {
		return c.Name[len("group."):]
	}
	return ""
}

// GetPrograms get the programs from the group
func (c *Entry) GetPrograms() []string {
	if c.IsGroup() {
		r := c.GetStringArray("programs", ",")
		for i, p := range r {
			r[i] = strings.TrimSpace(p)
		}
		return r
	}
	return make([]string, 0)
}

// String dump the configuration as string
func (c *Entry) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	fmt.Fprintf(buf, "configDir=%s\n", c.ConfigDir)
	fmt.Fprintf(buf, "group=%s\n", c.Group)
	for k, v := range c.keyValues {
		fmt.Fprintf(buf, "%s=%s\n", k, v)
	}
	return buf.String()
}

// Config memory representations of supervisor configuration file
type Config struct {
	configFile string
	// mapping between the section name and the configure
	entries map[string]*Entry

	ProgramGroup *ProcessGroup
}

// NewEntry create a configuration entry
func NewEntry(configDir string) *Entry {
	return &Entry{configDir, "", "", make(map[string]string), nil}
}

// NewConfig create Config object
func NewConfig(configFile string) *Config {
	return &Config{configFile, make(map[string]*Entry), NewProcessGroup()}
}

// create a new entry or return the already-exist entry
func (c *Config) createEntry(name, configDir string) *Entry {
	entry, ok := c.entries[name]

	if !ok {
		entry = NewEntry(configDir)
		c.entries[name] = entry
	}
	return entry
}

//
// Load load the configuration and return the loaded programs
func (c *Config) Load() ([]string, error) {
	ii := Ini{c.configFile}
	return ii.Load(c)
}

// GetConfigFileDir get the directory of supervisor configuration file
func (c *Config) GetConfigFileDir() string {
	return filepath.Dir(c.configFile)
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

// GetUnixHTTPServer get the unix_http_server section
func (c *Config) GetUnixHTTPServer() (*Entry, bool) {
	entry, ok := c.entries["unix_http_server"]

	return entry, ok
}

// GetInetHTTPServer Get the inet_http_server configuration section
func (c *Config) GetInetHTTPServer() (*Entry, bool) {
	entry, ok := c.entries["inet_http_server"]
	return entry, ok
}

// GetSupervisorctl Get the "supervisorctl" section
func (c *Config) GetSupervisorctl() (*Entry, bool) {
	entry, ok := c.entries["supervisorctl"]
	return entry, ok
}

// GetEntries get the configuration entries by filter
func (c *Config) GetEntries(filterFunc func(entry *Entry) bool) []*Entry {
	result := make([]*Entry, 0)
	for _, entry := range c.entries {
		if filterFunc(entry) {
			result = append(result, entry)
		}
	}
	return result
}

// GetGroups get entries of all the program groups
func (c *Config) GetGroups() []*Entry {
	return c.GetEntries(func(entry *Entry) bool {
		return entry.IsGroup()
	})
}

// GetPrograms get entries of all programs
func (c *Config) GetPrograms() []*Entry {
	programs := c.GetEntries(func(entry *Entry) bool {
		return entry.IsProgram()
	})

	return sortProgram(programs)
}

// GetProgramNames get all the program names
func (c *Config) GetProgramNames() []string {
	result := make([]string, 0)
	programs := c.GetPrograms()

	programs = sortProgram(programs)
	for _, entry := range programs {
		result = append(result, entry.GetProgramName())
	}
	return result
}

// GetProgram return the proram configure entry or nil
func (c *Config) GetProgram(name string) *Entry {
	for _, entry := range c.entries {
		if entry.IsProgram() && entry.GetProgramName() == name {
			return entry
		}
	}
	return nil
}

// GetBool get value of key as bool
func (c *Entry) GetBool(key string, defValue bool) bool {
	value, ok := c.keyValues[key]

	if ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defValue
}

// HasParameter check if has parameter
func (c *Entry) HasParameter(key string) bool {
	_, ok := c.keyValues[key]
	return ok
}

func toInt(s string, factor, defValue int) int {
	i, err := strconv.Atoi(s)
	if err == nil {
		return i * factor
	}
	return defValue
}

// GetInt get the value of the key as int
func (c *Entry) GetInt(key string, defValue int) int {
	value, ok := c.keyValues[key]

	if ok {
		return toInt(value, 1, defValue)
	}
	return defValue
}

// GetEnv get the value of key as environment setting. An environment string example:
//  environment = A="env 1",B="this is a test"
func (c *Entry) GetEnv(key string) []string {
	value, ok := c.keyValues[key]
	env := make([]string, 0)

	if ok {
		start := 0
		n := len(value)
		var i int
		for {
			for i = start; i < n && value[i] != '='; {
				i++
			}
			key := value[start:i]
			start = i + 1
			if value[start] == '"' {
				for i = start + 1; i < n && value[i] != '"'; {
					i++
				}
				if i < n {
					env = append(env, fmt.Sprintf("%s=%s", strings.TrimSpace(key), strings.TrimSpace(value[start+1:i])))
				}
				if i+1 < n && value[i+1] == ',' {
					start = i + 2
				} else {
					break
				}
			} else {
				for i = start; i < n && value[i] != ','; {
					i++
				}
				if i < n {
					env = append(env, fmt.Sprintf("%s=%s", strings.TrimSpace(key), strings.TrimSpace(value[start:i])))
					start = i + 1
				} else {
					env = append(env, fmt.Sprintf("%s=%s", strings.TrimSpace(key), strings.TrimSpace(value[start:])))
					break
				}
			}
		}
	}

	result := make([]string, 0)
	for i := 0; i < len(env); i++ {
		tmp, err := NewStringExpression("program_name", c.GetProgramName(),
			"process_num", c.GetString("process_num", "0"),
			"group_name", c.GetGroupName(),
			"here", c.ConfigDir).Eval(env[i])
		if err == nil {
			result = append(result, tmp)
		}
	}
	return result
}

// GetString get the value of key as string
func (c *Entry) GetString(key, defValue string) string {
	s, ok := c.keyValues[key]

	if ok {
		env := NewStringExpression("here", c.ConfigDir)
		repS, err := env.Eval(s)
		if err == nil {
			return repS
		}
		zap.L().Warn("Unable to parse expression", zap.Error(err), zap.String("program", c.GetProgramName()), zap.String("key", key))
	}
	return defValue
}

// GetStringExpression get the value of key as string and attempt to parse it with StringExpression
func (c *Entry) GetStringExpression(key, defValue string) string {
	s, ok := c.keyValues[key]
	if !ok || s == "" {
		return ""
	}

	hostName, err := os.Hostname()
	if err != nil {
		hostName = "Unknown"
	}
	result, err := NewStringExpression("program_name", c.GetProgramName(),
		"process_num", c.GetString("process_num", "0"),
		"group_name", c.GetGroupName(),
		"here", c.ConfigDir,
		"host_node_name", hostName).Eval(s)
	if err != nil {
		zap.L().Warn("Unable to parse expression", zap.Error(err), zap.String("program", c.GetProgramName()), zap.String("key", key))
		return s
	}

	return result
}

// GetStringArray get the string value and split it as array with "sep"
func (c *Entry) GetStringArray(key, sep string) []string {
	s, ok := c.keyValues[key]

	var parts []string
	if ok {
		parts = strings.Split(s, sep)
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
	}
	return parts
}

// GetBytes get the value of key as the bytes setting.
//
//	logSize=1MB
//	logSize=1GB
//	logSize=1KB
//	logSize=1024
//
func (c *Entry) GetBytes(key string, defValue int) int {
	v, ok := c.keyValues[key]

	if ok {
		if len(v) > 2 {
			lastTwoBytes := v[len(v)-2:]
			if lastTwoBytes == "MB" {
				return toInt(v[:len(v)-2], 1024*1024, defValue)
			} else if lastTwoBytes == "GB" {
				return toInt(v[:len(v)-2], 1024*1024*1024, defValue)
			} else if lastTwoBytes == "KB" {
				return toInt(v[:len(v)-2], 1024, defValue)
			}
		}
		return toInt(v, 1, defValue)
	}
	return defValue
}

// String convert the configuration to string represents
func (c *Config) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	fmt.Fprintf(buf, "configFile:%s\n", c.configFile)
	for k, v := range c.entries {
		fmt.Fprintf(buf, "[program.%s]\n", k)
		fmt.Fprintf(buf, "%s\n", v.String())
	}
	return buf.String()
}

// RemoveProgram remove a program entry by its name
func (c *Config) RemoveProgram(programName string) {
	delete(c.entries, programName)
	c.ProgramGroup.Remove(programName)
}
