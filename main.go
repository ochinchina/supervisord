package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"os"
	"os/signal"
	"syscall"
)

type Options struct {
	Configuration string `short:"c" long:"configuration" description:"the configuration file" default:"supervisord.conf"`
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func initSignals(s *Supervisor) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.WithFields(log.Fields{"signal": sig}).Info("receive a signal to stop all process & xit")
		s.procMgr.ForEachProcess(func(proc *Process) {
			proc.Stop(true)
		})
		os.Exit(-1)
	}()

}

func main() {
	var options Options
	var parser = flags.NewParser(&options, flags.Default)
	parser.Parse()
	s := NewSupervisor(options.Configuration)
	initSignals(s)
	if err := s.Reload(); err != nil {
		panic(err)
	}
}
