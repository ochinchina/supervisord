package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Configuration string `short:"c" long:"configuration" description:"the configuration file" default:"supervisord.conf"`
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
	if err := s.Reload(); err != nil {
		panic(err)
	}
}
