package main

import (
	"io"
	"os"
)

var configTemplate = `[unix_http_server]
file=/tmp/supervisord.sock
username=test1
password={SHA}82ab876d1387bfafe46cc1c8a2ef074eae50cb1d

[inet_http_server]
port=127.0.0.1:9001
username=test1
password=thepassword

[program.x]
command=/bin/cat
process_name=%(program_name)s
autostart=true
startsecs=3
startretries=3
autorestart=true
exitcodes=0,2
stopsignal=TERM
stopwaitsecs=10
stopasgroup=true
killasgroup=true
user=user1
redirect_stderr=false
stdout_logfile=AUTO
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=10
stderr_logfile=AUTO
stderr_logfile_maxbytes=50MB
stderr_logfile_backups=10
environment=KEY="val",KEY2="val2"
directory=/tmp
serverurl=AUTO

[include]
files=/an/absolute/filename.conf /an/absolute/*.conf foo.conf config??.conf

[group.x]
programs=bar,baz
priority=999

[supervisorctl]
serverurl = unix:///tmp/supervisor.sock
username = chris
password = 123
`

// InitTemplateCommand implemnts flags.Commander interface
type InitTemplateCommand struct {
	OutFile string `short:"o" long:"output" description:"the output file name" required:"true"`
}

var initTemplateCommand InitTemplateCommand

// Execute execute the init command
func (x *InitTemplateCommand) Execute(args []string) error {
	f, err := os.Create(x.OutFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return GenTemplate(f)
}

// GenTemplate generate the template
func GenTemplate(writer io.Writer) error {
	_, err := writer.Write([]byte(configTemplate))
	return err
}

func init() {
	parser.AddCommand("init",
		"initialize a template",
		"The init subcommand writes the supported configurations to specified file",
		&initTemplateCommand)
}
