package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"unicode"

	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

type Options struct {
	Configuration string `short:"c" long:"configuration" description:"the configuration file" default:"supervisord.conf"`
	Daemon        bool   `short:"d" long:"daemon" description:"run as daemon"`
	EnvFile       string `long:"env-file" description:"the environment file"`
}

func init() {
	log.SetOutput(os.Stdout)
	if runtime.GOOS == "windows" {
		log.SetFormatter(&log.TextFormatter{DisableColors: true, FullTimestamp: true})
	} else {
		log.SetFormatter(&log.TextFormatter{DisableColors: false, FullTimestamp: true})
	}
	log.SetLevel(log.DebugLevel)
}

func initSignals(s *Supervisor) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.WithFields(log.Fields{"signal": sig}).Info("receive a signal to stop all process & exit")
		s.procMgr.StopAllProcesses()
		os.Exit(-1)
	}()

}

var options Options
var parser = flags.NewParser(&options, flags.Default & ^flags.PrintErrors)

func LoadEnvFile() {
	if len(options.EnvFile) <= 0 {
		return
	}
	f, err := os.Open(options.EnvFile)
	if err != nil {
		log.WithFields(log.Fields{"file": options.EnvFile}).Error("Fail to open environment file")
		return
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "export") && len(line) > len("export") && unicode.IsSpace(rune(line[len("export")])) {
			line = strings.TrimSpace(line[len("export"):])
		}
		pos := strings.Index(line, "=")
		if pos != -1 {
			k := strings.TrimSpace(line[0:pos])
			v := strings.TrimSpace(line[pos+1:])
			if len(k) > 0 && len(v) > 0 {
				os.Setenv(k, v)
			}
		}
	}
}

func RunServer() {
	// infinite loop for handling Restart ('reload' command)
	LoadEnvFile()
	for true {
		s := NewSupervisor(options.Configuration)
		initSignals(s)
		if sErr, _, _, _ := s.Reload(); sErr != nil {
			panic(sErr)
		}
		s.WaitForExit()
	}
}

func main() {
	if _, err := parser.Parse(); err != nil {
		flagsErr, ok := err.(*flags.Error)
		if ok {
			switch flagsErr.Type {
			case flags.ErrHelp:
				fmt.Fprintln(os.Stdout, err)
				os.Exit(0)
			case flags.ErrCommandRequired:
				if options.Daemon {
					Deamonize(RunServer)
				} else {
					RunServer()
				}
			default:
				panic(err)
			}
		}
	}
}
