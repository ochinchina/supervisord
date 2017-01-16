package main

import (
	"testing"
)


func TestEmptyCommandLine( t* testing.T ) {
	args, err := parseCommand( " ")
	if err == nil || len( args ) > 0 {
		t.Error( "fail to parse the empty command line")
	}
}

func TestNormalCommandLine(t *testing.T ) {
	args, err := parseCommand( "program arg1 arg2")
	if err != nil {
		t.Error( "fail to parse normal command line")
	}
	if args[0] != "program" || args[1] != "arg1" || args[2] != "arg2" {
		t.Error( "fail to parse normal command line" )
	}
}

func TestCommandLineWithQuotationMarks(t* testing.T ) {
	args, err := parseCommand( "program 'this is arg1' args=\"this is arg2\"" )
	if err != nil || len( args) != 3 {
		t.Error( "fail to parse command line with quotation marks")
	}
	if args[0] != "program" || args[1] != "this is arg1" || args[2] != "args=\"this is arg2\"" {
		t.Error( "fail to parse command line with quotation marks")
	}
}
