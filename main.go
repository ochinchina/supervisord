package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"os"
)

type Options struct {
	Configuration string `short:"c" long:"configuration" description:"the configuration file" optional:"yes" default:"supervisord.conf"`
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	var options Options
	var parser = flags.NewParser(&options, flags.Default)
	parser.Parse()
	s := NewSupervisor(options.Configuration)
	s.Reload()

}
