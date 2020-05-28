package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode"

	"github.com/stuartcarnie/gopm"
	"github.com/stuartcarnie/gopm/internal/zap/encoder"
	"github.com/stuartcarnie/gopm/process"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "term-color"
	cfg.DisableStacktrace = true
	cfg.EncoderConfig = encoder.NewDevelopmentEncoderConfig()
	cfg.EncoderConfig.CallerKey = ""
	log, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(log)
}

func initSignals(s *gopm.Supervisor) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		zap.L().Info("Received signal to stop all processes and exit", zap.Stringer("signal", sig))
		s.GetManager().StopAllProcesses()
		os.Exit(-1)
	}()
}

func loadEnvFile() {
	if len(rootOpt.EnvFile) <= 0 {
		return
	}
	// try to open the rootOpt file
	f, err := os.Open(rootOpt.EnvFile)
	if err != nil {
		zap.L().Error("Failed to open environment file", zap.String("file", rootOpt.EnvFile))
		return
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		// for each line
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		// if line starts with '#', it is a comment line, ignore it
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] == '#' {
			continue
		}
		// if environment variable is exported with "export"
		if strings.HasPrefix(line, "export") && len(line) > len("export") && unicode.IsSpace(rune(line[len("export")])) {
			line = strings.TrimSpace(line[len("export"):])
		}
		// split the environment variable with "="
		pos := strings.Index(line, "=")
		if pos != -1 {
			k := strings.TrimSpace(line[0:pos])
			v := strings.TrimSpace(line[pos+1:])
			// if key and value are not empty, put it into the environment
			if len(k) > 0 && len(v) > 0 {
				os.Setenv(k, v)
			}
		}
	}
}

func runServer() {
	// infinite loop for handling Restart ('reload' command)
	loadEnvFile()
	for {
		s := gopm.NewSupervisor(rootOpt.Configuration)
		initSignals(s)
		if _, _, _, sErr := s.Reload(); sErr != nil {
			panic(sErr)
		}
		s.WaitForExit()
	}
}

var (
	rootOpt = struct {
		Configuration string
		EnvFile       string
		Shell         string
	}{}

	rootCmd = cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			process.SetShellArgs(strings.Split(rootOpt.Shell, " "))
			runServer()
		},
	}
)

func getDefaultShell() string {
	sh := os.Getenv("SHELL")
	if sh == "" {
		return "/bin/sh -c"
	}

	return sh + " -c"
}

func main() {
	gopm.ReapZombie()

	rootCmd.PersistentFlags().StringVarP(&rootOpt.Configuration, "config", "c", "", "Configuration file")
	flags := rootCmd.Flags()
	flags.StringVar(&rootOpt.EnvFile, "env-file", "", "An optional environment file")
	flags.StringVar(&rootOpt.Shell, "shell", getDefaultShell(), "Specify an alternate shell path")
	_ = rootCmd.MarkFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to execute command", err)
	}
}
