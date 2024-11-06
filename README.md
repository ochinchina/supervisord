# Go Implementation of Supervisord

[![Go Report Card](https://goreportcard.com/badge/github.com/QPod/supervisord)](https://goreportcard.com/report/github.com/QPod/supervisord)

## Why this project?

The Python version [supervisord](http://supervisord.org/) is a widely-used and powerful tool to manage the processes, yet the python environment required by it, in some situation, for example in the docker environment, is too big for us (same for `pm2` and NodeJS).

This project re-implements supervisord in golang. Compiled `supervisord` has a very small footprint (<5MB), which is suitable for environments where python is not available.

## Download the supervisord binary

You can download the binary of supervisord from the [GitHub Release page](https://github.com/QPod/supervisord/releases/), or use the following command.

```shell
   OS="linux" && ARCH="amd64" \
&& VER_SUPERVISORD=$(curl -sL https://github.com/QPod/supervisord/releases.atom | grep "releases/tag" | head -1 | grep -Po '(\d[\d|.]+)') \
&& URL_SUPERVISORD="https://github.com/QPod/supervisord/releases/download/v${VER_SUPERVISORD}/supervisord_${VER_SUPERVISORD}_${OS}_${ARCH}.tar.gz" \
&& echo "Downloading Supervisord ${VER_SUPERVISORD} from ${URL_SUPERVISORD}" \
&& curl -o /tmp/TMP.tgz -sL $URL_SUPERVISORD && tar -C ./ -xzf /tmp/TMP.tgz && rm /tmp/TMP.tgz \
&& ./supervisord version
```

## Building the supervisord

You can compile `supervisord` using the following commands on Linux:

0. Have golang 1.22+ installed.
1. `go mod tidy`
2. `GOOS=linux go build -tags release -a -ldflags "-linkmode external -extldflags -static" -o supervisord`
3. Optionally, use [upx](https://github.com/upx/upx/) to reduce the executable file size.

## Run the supervisord

After a supervisord binary has been generated, create a supervisord configuration file and start the supervisord like this:

```Shell
$ cat supervisor.conf
[program:test]
command = /your/program args

$ supervisord -c supervisor.conf
```

Please note that config-file location autodetected in this order:

1. $CWD/supervisord.conf
2. $CWD/etc/supervisord.conf
3. /etc/supervisord.conf
4. /etc/supervisor/supervisord.conf (since Supervisor 3.3.0)
5. ../etc/supervisord.conf (Relative to the executable)
6. ../supervisord.conf (Relative to the executable)

### Run as daemon with web-ui

Add the inet interface in your configuration, and then run `supervisord -c supervisor.conf -d`:

```ini
[inet_http_server]
port=127.0.0.1:9001
```

In order to manage the daemon, you can use `supervisord ctl` subcommand, available subcommands are: `status`, `start`, `stop`, `shutdown`, `reload`.

```shell
supervisord ctl status
supervisord ctl status program-1 program-2...
supervisord ctl status group:*
supervisord ctl stop program-1 program-2...
supervisord ctl stop group:*
supervisord ctl stop all
supervisord ctl start program-1 program-2...
supervisord ctl start group:*
supervisord ctl start all
supervisord ctl shutdown
supervisord ctl reload
supervisord ctl signal <signal_name> <process_name> <process_name> ...
supervisord ctl signal all
supervisord ctl pid <process_name>
supervisord ctl fg <process_name>
```

Please note that `supervisor ctl` subcommand works correctly only if http server is enabled in [inet_http_server], and **serverurl** correctly set. Unix domain socket is not currently supported for this pupose.

Serverurl parameter detected in the following order:

- check if option -s or --serverurl is present, use this url
- check if -c option is present, and the "serverurl" in "supervisorctl" section is present, use "serverurl" in section "supervisorctl"
- check if "serverurl" in section "supervisorctl" is defined in autodetected supervisord.conf-file location and if it is - use found value
- use http://localhost:9001

## Check the version

Command "version" will show the current supervisord binary version: `supervisord version`.

## Supported features

### Http server

Http server can work via both unix domain socket and TCP. Basic auth is optional and supported too.

- The unix domain socket setting is in the `unix_http_server` section.
- The TCP http server setting is in `inet_http_server` section.

If neither `inet_http_server` nor `unix_http_server` is configured in the configuration file, no http server will be started.

## Config File Guideline

Please refer to [detailed documentations](./doc/doc-config.md).
