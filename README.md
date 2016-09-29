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
$ supervisord supervisor.conf
```
