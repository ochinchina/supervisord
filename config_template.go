package main

import (
	"io"
	"os"
)

var configTemplate = `[unix_http_server]
file=/tmp/supervisord.sock
#chmod=not support
#chown=not support
username=test1
password={SHA}82ab876d1387bfafe46cc1c8a2ef074eae50cb1d

[inet_http_server]
port=127.0.0.1:9001
username=test1
password=thepassword

[supervisord]
logfile=%(here)s/supervisord.log
logfileMaxbytes=50MB
logfileBackups=10
loglevel=info
pidfile=%(here)s/supervisord.pid
#umask=not support
#nodaemon=not support
#minfds=not support
#minprocs=not support
#nocleanup=not support
#childlogdir=not support
#user=not support
#directory=not support
#strip_ansi=not support
#environment=not support
identifier=supervisor

[program:x]
command=/bin/cat
process_name=%(program_name)s
numprocs=1
#numprocs_start=not support
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
stdout_capture_maxbytes=0
stdout_events_enabled=true
stderr_logfile=AUTO
stderr_logfile_maxbytes=50MB
stderr_logfile_backups=10
stderr_capture_maxbytes=0
stderr_events_enabled=false
environment=KEY="val",KEY2="val2"
envFiles=global.env,prod.env
directory=/tmp
#umask=not support
serverurl=AUTO

[include]
files=/an/absolute/filename.conf /an/absolute/*.conf foo.conf config??.conf

[group:x]
programs=bar,baz
priority=999

[eventlistener:x]
command=/bin/eventlistener
process_name=%(program_name)s
numprocs=1
#numprocs_start=not support
autostart=true
startsecs=3
startretries=3
autorestart=true
exitcodes=0,2
stopsignal=TERM
stopwaitsecs=10
#stopasgroup=not support
#killasgroup=not support
user=user1
redirect_stderr=false
stdout_logfile=AUTO
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=10
stdout_capture_maxbytes=0
stdout_events_enabled=true
stderr_logfile=AUTO
stderr_logfile_maxbytes=50MB
stderr_logfile_backups=10
stderr_capture_maxbytes=0
stderr_events_enabled=false
environment=KEY="val",KEY2="val2"
envFiles=global.env,prod.env
directory=/tmp
#umask=not support
serverurl=AUTO
buffer_size=10240
events=PROCESS_STATE
#result_handler=not support

[supervisorctl]
serverurl = unix:///tmp/supervisor.sock
username = chris
password = 123
#prompt = not support
`

// InitTemplateCommand implements flags.Commander interface
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
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
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
