package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-envparse"
	"github.com/ochinchina/go-ini"
	log "github.com/sirupsen/logrus"
)

// Entry standards for a configuration section in supervisor configuration file
type Entry struct {
	ConfigDir string
	Group     string
	Name      string
	keyValues map[string]string
}

// IsProgram returns true if this is a program section
func (c *Entry) IsProgram() bool {
	return strings.HasPrefix(c.Name, "program:")
}

// GetProgramName returns program name
func (c *Entry) GetProgramName() string {
	if strings.HasPrefix(c.Name, "program:") {
		return c.Name[len("program:"):]
	}
	return ""
}

// IsEventListener returns true if this section is for event listener
func (c *Entry) IsEventListener() bool {
	return strings.HasPrefix(c.Name, "eventlistener:")
}

// GetEventListenerName returns event listener name
func (c *Entry) GetEventListenerName() string {
	if strings.HasPrefix(c.Name, "eventlistener:") {
		return c.Name[len("eventlistener:"):]
	}
	return ""
}

// IsGroup returns true if it is group section
func (c *Entry) IsGroup() bool {
	return strings.HasPrefix(c.Name, "group:")
}

// GetGroupName returns group name if entry is a group
func (c *Entry) GetGroupName() string {
	if strings.HasPrefix(c.Name, "group:") {
		return c.Name[len("group:"):]
	}
	return ""
}

// GetPrograms returns slice with programs from the group
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

func (c *Entry) setGroup(group string) {
	c.Group = group
}

// String dumps configuration as a string
func (c *Entry) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	for k, v := range c.keyValues {
		fmt.Fprintf(buf, "%s=%s\n", k, v)
	}
	return buf.String()
}

// Config memory representation of supervisor configuration file
type Config struct {
	configFile string
	// mapping between the section name and configuration entry
	entries map[string]*Entry

	ProgramGroup *ProcessGroup
}

// NewEntry creates configuration entry
func NewEntry(configDir string) *Entry {
	return &Entry{configDir, "", "", make(map[string]string)}
}

// NewConfig creates Config object
func NewConfig(configFile string) *Config {
	return &Config{configFile, make(map[string]*Entry), NewProcessGroup()}
}

// create a new entry or return the already-exist entry
func (c *Config) createEntry(name string, configDir string) *Entry {
	entry, ok := c.entries[name]

	if !ok {
		entry = NewEntry(configDir)
		c.entries[name] = entry
	}
	return entry
}

//
// Load the configuration and return loaded programs
func (c *Config) Load() ([]string, error) {
	myini := ini.NewIni()
	c.ProgramGroup = NewProcessGroup()
	log.WithFields(log.Fields{"file": c.configFile}).Info("load configuration from file")
	myini.LoadFile(c.configFile)

	includeFiles := c.getIncludeFiles(myini)
	for _, f := range includeFiles {
		log.WithFields(log.Fields{"file": f}).Info("load configuration from file")
		myini.LoadFile(f)
	}
	return c.parse(myini), nil
}

func (c *Config) getIncludeFiles(cfg *ini.Ini) []string {
	result := make([]string, 0)
	if includeSection, err := cfg.GetSection("include"); err == nil {
		key, err := includeSection.GetValue("files")
		if err == nil {
			env := NewStringExpression("here", c.GetConfigFileDir())
			files := strings.Fields(key)
			for _, fRaw := range files {
				dir := c.GetConfigFileDir()
				f, err := env.Eval(fRaw)
				if err != nil {
					continue
				}
				if filepath.IsAbs(f) {
					dir = filepath.Dir(f)
				} else {
					dir = filepath.Join(c.GetConfigFileDir(), filepath.Dir(f))
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
	}
	return result
}

func (c *Config) parse(cfg *ini.Ini) []string {
	c.setProgramDefaultParams(cfg)
	c.parseGroup(cfg)
	loadedPrograms := c.parseProgram(cfg)

	// parse non-group, non-program and non-eventlistener sections
	for _, section := range cfg.Sections() {
		if !strings.HasPrefix(section.Name, "group:") && !strings.HasPrefix(section.Name, "program:") && !strings.HasPrefix(section.Name, "eventlistener:") {
			entry := c.createEntry(section.Name, c.GetConfigFileDir())
			c.entries[section.Name] = entry
			entry.parse(section)
		}
	}
	return loadedPrograms
}

// set the default parameters of programs
func (c *Config) setProgramDefaultParams(cfg *ini.Ini) {
	programDefaultSection, err := cfg.GetSection("program-default")
	if err == nil {
		for _, section := range cfg.Sections() {
			if section.Name == "program-default" || !strings.HasPrefix(section.Name, "program:") {
				continue
			}
			for _, key := range programDefaultSection.Keys() {
				if !section.HasKey(key.Name()) {
					section.Add(key.Name(), key.ValueWithDefault(""))
				}
			}

		}
	}
}

// GetConfigFileDir returns directory of supervisord configuration file
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

// GetUnixHTTPServer returns unix_http_server configuration section
func (c *Config) GetUnixHTTPServer() (*Entry, bool) {
	entry, ok := c.entries["unix_http_server"]

	return entry, ok
}

// GetSupervisord returns "supervisord" configuration section
func (c *Config) GetSupervisord() (*Entry, bool) {
	entry, ok := c.entries["supervisord"]
	return entry, ok
}

// GetInetHTTPServer returns inet_http_server configuration section
func (c *Config) GetInetHTTPServer() (*Entry, bool) {
	entry, ok := c.entries["inet_http_server"]
	return entry, ok
}

// GetSupervisorctl returns "supervisorctl" configuration section
func (c *Config) GetSupervisorctl() (*Entry, bool) {
	entry, ok := c.entries["supervisorctl"]
	return entry, ok
}

// GetEntries returns configuration entries by filter
func (c *Config) GetEntries(filterFunc func(entry *Entry) bool) []*Entry {
	result := make([]*Entry, 0)
	for _, entry := range c.entries {
		if filterFunc(entry) {
			result = append(result, entry)
		}
	}
	return result
}

// GetGroups returns configuration entries of all program groups
func (c *Config) GetGroups() []*Entry {
	return c.GetEntries(func(entry *Entry) bool {
		return entry.IsGroup()
	})
}

// GetPrograms returns configuration entries of all programs
func (c *Config) GetPrograms() []*Entry {
	programs := c.GetEntries(func(entry *Entry) bool {
		return entry.IsProgram()
	})

	return sortProgram(programs)
}

// GetEventListeners returns configuration entries of event listeners
func (c *Config) GetEventListeners() []*Entry {
	eventListeners := c.GetEntries(func(entry *Entry) bool {
		return entry.IsEventListener()
	})

	return eventListeners
}

// GetProgramNames returns slice with all program names
func (c *Config) GetProgramNames() []string {
	result := make([]string, 0)
	programs := c.GetPrograms()

	programs = sortProgram(programs)
	for _, entry := range programs {
		result = append(result, entry.GetProgramName())
	}
	return result
}

// GetProgram returns the program configuration entry or nil
func (c *Config) GetProgram(name string) *Entry {
	for _, entry := range c.entries {
		if entry.IsProgram() && entry.GetProgramName() == name {
			return entry
		}
	}
	return nil
}

// GetBool gets value of key as bool
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

// HasParameter checks if key (parameter) has value
func (c *Entry) HasParameter(key string) bool {
	_, ok := c.keyValues[key]
	return ok
}

func toInt(s string, factor int, defValue int) int {
	i, err := strconv.Atoi(s)
	if err == nil {
		return i * factor
	}
	return defValue
}

// GetInt gets value of the key as int
func (c *Entry) GetInt(key string, defValue int) int {
	value, ok := c.keyValues[key]

	if ok {
		return toInt(value, 1, defValue)
	}
	return defValue
}

func parseEnv(s string) *map[string]string {
	result := make(map[string]string)
	start := 0
	n := len(s)
	var i int
	for {
		// find the '='
		for i = start; i < n && s[i] != '='; {
			i++
		}
		key := s[start:i]
		start = i + 1
		if s[start] == '"' {
			for i = start + 1; i < n && s[i] != '"'; {
				i++
			}
			if i < n {
				result[strings.TrimSpace(key)] = strings.TrimSpace(s[start+1 : i])
			}
			if i+1 < n && s[i+1] == ',' {
				start = i + 2
			} else {
				break
			}
		} else {
			for i = start; i < n && s[i] != ','; {
				i++
			}
			if i < n {
				result[strings.TrimSpace(key)] = strings.TrimSpace(s[start:i])
				start = i + 1
			} else {
				result[strings.TrimSpace(key)] = strings.TrimSpace(s[start:])
				break
			}
		}
	}

	return &result
}

func parseEnvFiles(s string) *map[string]string {
	result := make(map[string]string)
	for _, envFilePath := range strings.Split(s, ",") {
		envFilePath = strings.TrimSpace(envFilePath)
		f, err := os.Open(envFilePath)
		if err != nil {
			log.WithFields(log.Fields{
				log.ErrorKey: err,
				"file":       envFilePath,
			}).Error("Read file failed: " + envFilePath)
			continue
		}
		r, err := envparse.Parse(f)
		if err != nil {
			log.WithFields(log.Fields{
				log.ErrorKey: err,
				"file":       envFilePath,
			}).Error("Parse env file failed: " + envFilePath)
			continue
		}
		for k, v := range r {
			result[k] = v
		}
	}
	return &result
}

// GetEnv returns slice of strings with keys separated from values by single "=". An environment string example:
//  environment = A="env 1",B="this is a test"
func (c *Entry) GetEnv(key string) []string {
	value, ok := c.keyValues[key]
	result := make([]string, 0)

	if ok {
		for k, v := range *parseEnv(value) {
			tmp, err := NewStringExpression("program_name", c.GetProgramName(),
				"process_num", c.GetString("process_num", "0"),
				"group_name", c.GetGroupName(),
				"here", c.ConfigDir).Eval(fmt.Sprintf("%s=%s", k, v))
			if err == nil {
				result = append(result, tmp)
			}
		}
	}

	return result
}

// GetEnvFromFiles returns slice of strings with keys separated from values by single "=". An envFile example:
//  envFiles = global.env,prod.env
// cat global.env
// varA=valueA
func (c *Entry) GetEnvFromFiles(key string) []string {
	value, ok := c.keyValues[key]
	result := make([]string, 0)

	if ok {
		for k, v := range *parseEnvFiles(value) {
			tmp, err := NewStringExpression("program_name", c.GetProgramName(),
				"process_num", c.GetString("process_num", "0"),
				"group_name", c.GetGroupName(),
				"here", c.ConfigDir).Eval(fmt.Sprintf("%s=%s", k, v))
			if err == nil {
				result = append(result, tmp)
			}
		}
	}

	return result
}

// GetString returns value of the key as a string
func (c *Entry) GetString(key string, defValue string) string {
	s, ok := c.keyValues[key]

	if ok {
		env := NewStringExpression("here", c.ConfigDir)
		repS, err := env.Eval(s)
		if err == nil {
			return repS
		}
		log.WithFields(log.Fields{
			log.ErrorKey: err,
			"program":    c.GetProgramName(),
			"key":        key,
		}).Warn("Unable to parse expression")
	}
	return defValue
}

// GetStringExpression returns value of key as a string and attempts to parse it with StringExpression
func (c *Entry) GetStringExpression(key string, defValue string) string {
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
		log.WithFields(log.Fields{
			log.ErrorKey: err,
			"program":    c.GetProgramName(),
			"key":        key,
		}).Warn("unable to parse expression")
		return s
	}

	return result
}

// GetStringArray gets string value and split it with "sep" to slice
func (c *Entry) GetStringArray(key string, sep string) []string {
	s, ok := c.keyValues[key]

	if ok {
		return strings.Split(s, sep)
	}
	return make([]string, 0)
}

// GetBytes returns value of the key as bytes setting.
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

func (c *Entry) parse(section *ini.Section) {
	c.Name = section.Name
	for _, key := range section.Keys() {
		c.keyValues[key.Name()] = strings.TrimSpace(key.ValueWithDefault(""))
	}
}

func (c *Config) parseGroup(cfg *ini.Ini) {

	// parse the group at first
	for _, section := range cfg.Sections() {
		if strings.HasPrefix(section.Name, "group:") {
			entry := c.createEntry(section.Name, c.GetConfigFileDir())
			entry.parse(section)
			groupName := entry.GetGroupName()
			programs := entry.GetPrograms()
			for _, program := range programs {
				c.ProgramGroup.Add(groupName, program)
			}
		}
	}
}

func (c *Config) isProgramOrEventListener(section *ini.Section) (bool, string) {
	// check if it is a program or event listener section
	isProgram := strings.HasPrefix(section.Name, "program:")
	isEventListener := strings.HasPrefix(section.Name, "eventlistener:")
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
func (c *Config) parseProgram(cfg *ini.Ini) []string {
	loadedPrograms := make([]string, 0)
	for _, section := range cfg.Sections() {
		programOrEventListener, prefix := c.isProgramOrEventListener(section)

		// if it is program or event listener
		if programOrEventListener {
			// get the number of processes
			numProcs, err := section.GetInt("numprocs")
			programName := section.Name[len(prefix):]
			if err != nil {
				numProcs = 1
			}
			procName, err := section.GetValue("process_name")
			if numProcs > 1 {
				if err != nil || strings.Index(procName, "%(process_num)") == -1 {
					log.WithFields(log.Fields{
						"numprocs":     numProcs,
						"process_name": procName,
					}).Error("no process_num in process name")
				}
			}
			originalProcName := programName
			if err == nil {
				originalProcName = procName
			}

			originalCmd := section.GetValueWithDefault("command", "")

			for i := 1; i <= numProcs; i++ {
				envs := NewStringExpression("program_name", programName,
					"process_num", fmt.Sprintf("%d", i),
					"group_name", c.ProgramGroup.GetGroup(programName, programName),
					"here", c.GetConfigFileDir())
				envValue, err := section.GetValue("environment")
				if err == nil {
					for k, v := range *parseEnv(envValue) {
						envs.Add(fmt.Sprintf("ENV_%s", k), v)
					}
				}
				cmd, err := envs.Eval(originalCmd)
				if err != nil {
					log.WithFields(log.Fields{
						log.ErrorKey: err,
						"program":    programName,
					}).Error("get envs failed")
					continue
				}
				section.Add("command", cmd)

				procName, err := envs.Eval(originalProcName)
				if err != nil {
					log.WithFields(log.Fields{
						log.ErrorKey: err,
						"program":    programName,
					}).Error("get envs failed")
					continue
				}

				section.Add("process_name", procName)
				section.Add("numprocs_start", fmt.Sprintf("%d", i-1))
				section.Add("process_num", fmt.Sprintf("%d", i))
				entry := c.createEntry(procName, c.GetConfigFileDir())
				entry.parse(section)
				entry.Name = prefix + procName
				group := c.ProgramGroup.GetGroup(programName, programName)
				entry.Group = group
				loadedPrograms = append(loadedPrograms, procName)
			}
		}
	}
	return loadedPrograms
}

// String converts configuration to the string
func (c *Config) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	for _, v := range c.entries {
		fmt.Fprintf(buf, "[%s]\n", v.Name)
		fmt.Fprintf(buf, "%s\n", v.String())
	}
	return buf.String()
}

// RemoveProgram removes program entry by its name
func (c *Config) RemoveProgram(programName string) {
	delete(c.entries, programName)
	c.ProgramGroup.Remove(programName)
}
