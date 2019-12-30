[![Go Report Card](https://goreportcard.com/badge/github.com/ochinchina/supervisord)](https://goreportcard.com/report/github.com/ochinchina/supervisord)

# Why this project?

The python script supervisord is a powerful tool used by a lot of guys to manage the processes. I like the tool supervisord also.

But this tool requires us to install the big python environment. In some situation, for example in the docker environment, the python is too big for us.

In this project, the supervisord is re-implemented in go-lang. The compiled supervisord is very suitable for these environment that the python is not installed.

# Compile the supervisord

Before compiling the supervisord, make sure the go-lang is installed in your environement.

To compile the go-lang version supervisord, run following commands (required go 1.11+):

1. local: `go build`
1. linux: `env GOOS=linux GOARCH=amd64 go build -o supervisord_linux_amd64`

# Run the supervisord

After the supervisord binary is generated, create a supervisord configuration file and start the supervisord like below:

```shell
$ cat supervisor.conf
[program:test]
command = /your/program args
$ supervisord -c supervisor.conf
```
# Run as daemon
Add the inet interface in your configuration:
```ini
[inet_http_server]
port=127.0.0.1:9001
```
then run
```shell
$ supervisord -c supervisor.conf -d
```
In order to controll the daemon, you can use `$ supervisord ctl` subcommand, available commands are: `status`, `start`, `stop`, `shutdown`, `reload`.

```shell
$ supervisord ctl status
$ supervisord ctl status program-1 program-2...
$ supervisord ctl status group:*
$ supervisord ctl stop program-1 program-2...
$ supervisord ctl stop group:*
$ supervisord ctl stop all
$ supervisord ctl start program-1 program-2...
$ supervisord ctl start group:*
$ supervisord ctl start all
$ supervisord ctl shutdown
$ supervisord ctl reload
$ supervisord ctl signal <signal_name> <process_name> <process_name> ...
$ supervisord ctl signal all
$ supervisord ctl pid <process_name>
$ supervisord ctl fg <process_name>
```

the URL of supervisord in the "supervisor ctl" subcommand is dected in following order:

- check if option -s or --serverurl is present, use this url
- check if -c option is present and the "serverurl" in "supervisorctl" section is present, use the "serverurl" in section "supervisorctl"
- return http://localhost:9001

# Check the version

command "version" will show the current supervisor version.

```shell
$ supervisord version
```

# Supported features

## http server

the unix socket & TCP http server is supported. Basic auth is supported.

The unix socket setting is in the "unix_http_server" section.
The TCP http server setting is in "inet_http_server" section.

If both "inet_http_server" and "unix_http_server" is not configured in the configuration file, no http server will be started.

## supervisord information

The log & pid of supervisord process is supported by section "supervisord" setting.

## program

the following features is supported in the "program:x" section:

- program command
- process name
- numprocs
- numprocs_start
- autostart
- startsecs
- startretries
- autorestart
- exitcodes
- stopsignal
- stopwaitsecs
- stdout_logfile
- stdout_logfile_maxbytes
- stdout_logfile_backups
- redirect_stderr
- stderr_logfile
- stderr_logfile_maxbytes
- stderr_logfile_backups
- environment
- priority
- user
- directory
- stopasgroup
- killasgroup
- restartpause

### program extends

Following new keys are supported by the [program:xxx] section:

- **depends_on**: define program depends information. If program A depends on program B, C, the program B, C will be started before program A. Example:

```ini
[program:A]
depends_on = B, C

[program:B]
...
[program:C]
...
```

- **user**: user in the section "program:xxx" now is extended to support group with format "user[:group]". So "user" can be configured as:

```ini
[program:xxx]
user = user_name
...
```
or
```ini
[program:xxx]
user = user_name:group_name
...
```
- **stopsignal** list
one or more stop signal can be configured. If more than one stopsignal is configured, when stoping the program, the supervisor will send the signals to the program one by one with interval "stopwaitsecs". If the program does not exit after all the signals sent to the program, the supervisor will kill the program

- **restart_when_binary_changed**: a bool flag to control if the program should be restarted when the executable binary is changed

- **restart_directory_monitor**: a path to be monitored for restarting purpose
- **restart_file_pattern**: if a file is changed under restart_directory_monitor and the filename matches this pattern, the program will be restarted.

## Set default parameters for program

A section "program-default" is added and the default parameters for programs can be set in this section. This can reduce some parameters for programs. For example both test1 and test2 program have exactly same environment variables VAR1 and VAR2, the environment variable is decalred like:

```ini
[program:test1]
...
environment=VAR1="value1",VAR2="value2"

[program:test2]
...
environment=VAR1="value1",VAR2="value2"
```

the VAR1 and VAR2 environment variable can be moved to "program-default" section like:

```ini

[program-default]
environment=VAR1="value1",VAR2="value2"

[program:test1]
...

[program:test2]
...

```



## Group
the "group" section is supported and you can set "programs" item

## Events

the supervisor 3.x defined events are supported partially. Now it supports following events:

- all process state related events
- process communication event
- remote communication event
- tick related events
- process log related events

## Logs

The logs ( field stdout_logfile, stderr_logfile ) from programs managed by the supervisord can be written to:

```
- /dev/null, ignore the log
- /dev/stdout, write log to stdout
- /dev/stderr, write log to stderr
- syslog, write the log to local syslog
- syslog @[protocol:]host[:port], write the log to remote syslog. protocol must be "tcp" or "udp", if missing, "udp" will be used. If port is missing, for "udp" protocol, it's value is 514 and for "tcp" protocol, it's value is 6514.
- file name, write log to a file
```

Mutiple log file can be configured for the stdout_logfile and stderr_logfile with delimeter ',', for example if want to a program write log to both stdout and test.log file, the stdout_logfile for the program can be configured as:

```ini
stdout_logfile = test.log, /dev/stdout
```

# Web GUI

This supervisord has a default web GUI, you can start, stop & check the status of program from the GUI. Following picture shows the default web GUI:

![alt text](https://github.com/ochinchina/supervisord/blob/master/go_supervisord_gui.png)

# Usage from a Docker container

supervisord is compiled inside a Docker image to be used directly inside another image, from the Docker Hub version.

```Dockerfile
FROM debian:latest
COPY --from=ochinchina/supervisord:latest /usr/local/bin/supervisord /usr/local/bin/supervisord
CMD ["/usr/local/bin/supervisord"]
```

# The MIT License (MIT)

Copyright (c) <year> <copyright holders>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
