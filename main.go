package main

import(
	"os"
	"github.com/jessevdk/go-flags"
	log "github.com/Sirupsen/logrus"
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
	s := NewSupervisor( options.Configuration )
	s.Reload()


}

