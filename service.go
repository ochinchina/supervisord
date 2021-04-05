package main

import (
	"fmt"

	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
)

// ServiceCommand install/uninstall/start/stop supervisord service
type ServiceCommand struct {
}

var serviceCommand ServiceCommand

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

// Execute implement Execute() method defined in flags.Commander interface, executes the given command
func (sc ServiceCommand) Execute(args []string) error {
	if len(args) == 0 {
		showUsage()
		return nil
	}

	serviceArgs := make([]string, 0)
	if options.Configuration != "" {
		serviceArgs = append(serviceArgs, "--configuration="+options.Configuration)
	}
	if options.EnvFile != "" {
		serviceArgs = append(serviceArgs, "--env-file="+options.EnvFile)
	}

	svcConfig := &service.Config{
		Name:        "go-supervisord",
		DisplayName: "go-supervisord",
		Description: "Supervisord service in golang",
		Arguments:   serviceArgs,
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Error("service init failed", err)
		return err
	}

	action := args[0]
	switch action {
	case "install":
		err := s.Install()
		if err != nil {
			log.Error("Failed to install service go-supervisord: ", err)
			fmt.Println("Failed to install service go-supervisord: ", err)
			return err
		} else {
			fmt.Println("Succeed to install service go-supervisord")
		}
	case "uninstall":
		s.Stop()
		err := s.Uninstall()
		if err != nil {
			log.Error("Failed to uninstall service go-supervisord: ", err)
			fmt.Println("Failed to uninstall service go-supervisord: ", err)
			return err
		} else {
			fmt.Println("Succeed to uninstall service go-supervisord")
		}
	case "start":
		err := s.Start()
		if err != nil {
			log.Error("Failed to start service: ", err)
			fmt.Println("Failed to start service: ", err)
			return err
		} else {
			fmt.Println("Succeed to start service go-supervisord")
		}
	case "stop":
		err := s.Stop()
		if err != nil {
			log.Error("Failed to stop service: ", err)
			fmt.Println("Failed to stop service: ", err)
			return err
		} else {
			fmt.Println("Succeed to stop service go-supervisord")
		}

	default:
		showUsage()
	}

	return nil
}

func showUsage() {
	fmt.Println("uasge: supervisord service install/uninstall/start/stop")
}

func init() {
	parser.AddCommand("service",
		"install/uninstall/start/stop service",
		"install/uninstall/start/stop service",
		&serviceCommand)
}
