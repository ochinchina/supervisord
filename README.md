# why this project? 

The python script supervisord is a powerful tool used by a lot of guys to manage the processes. I like the tool supervisord also.

But this tool requires us to install the big python environment. In some situation, for example in the docker environment, the python is too big for us.

In this project, the supervisord is re-implemented in go-lang. The compiled supervisord is very suitable for these environment that the python is not installed.

# Compile the supervisord

Before compiling the supervisord, make sure the go-lang is installed in your environement.

To compile the go-lang version supervisord, run following commands:

```shell
$ mkdir ~/go-supervisor
$ export GOPATH=~/go-supervisor
$ go get github.com/ochinchina/supervisord
```

# Run the supervisord

After the supervisord binary is generated, create a supervisord configuration file and start the supervisord like below:

```
$ cat supervisor.conf
[program:test]
command = /your/program args
$ supervisord -c supervisor.conf
```

# features supported

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

## Group
the "group" section is supported and you can set "programs" item

## Events

the supervisor 3.x defined events are supported partially. Now it supports following events:

- all process state related events
- process communication event
- remote communication event
- tick related events
- process log related events

# The MIT License (MIT)

Copyright (c) <year> <copyright holders>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
