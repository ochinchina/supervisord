package process

import (
	"fmt"
	"strings"
	"testing"
)

func TestEmptyCommandLine(t *testing.T) {
	args, err := parseCommand(" ")
	if err == nil || len(args) > 0 {
		t.Error("fail to parse the empty command line")
	}
}

func TestNormalCommandLine(t *testing.T) {
	args, err := parseCommand("program arg1 arg2")
	if err != nil {
		t.Error("fail to parse normal command line")
	}
	if args[0] != "program" || args[1] != "arg1" || args[2] != "arg2" {
		t.Error("fail to parse normal command line")
	}
}

func TestCommandLineWithQuotationMarks(t *testing.T) {
	args, err := parseCommand("program 'this is arg1' args=\"this is arg2\"")
	fmt.Printf("%s\n", strings.Join(args, ","))
	if err != nil || len(args) != 3 {
		t.Error("fail to parse command line with quotation marks")
	}
	if args[0] != "program" || args[1] != "this is arg1" || args[2] != "args=\"this is arg2\"" {
		t.Error("fail to parse command line with quotation marks")
	}
}

func TestCommandLineArgsIsQuatationMarks(t *testing.T) {
	args, err := parseCommand("/home/test/nginx-1.13.0/objs/nginx -p /home/test/nginx-1.13.0 -c conf/nginx.conf -g \"daemon off;\"")
	fmt.Printf("%s\n", strings.Join(args, ","))
	if err != nil || len(args) != 7 {
		t.Error("fail to parse the command line")
	}
	if args[0] != "/home/test/nginx-1.13.0/objs/nginx" ||
		args[1] != "-p" ||
		args[2] != "/home/test/nginx-1.13.0" ||
		args[3] != "-c" ||
		args[4] != "conf/nginx.conf" ||
		args[5] != "-g" ||
		args[6] != "daemon off;" {
		t.Error("fail to parse command line")
	}
}
