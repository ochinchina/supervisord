package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/stuartcarnie/gopm"
	"github.com/stuartcarnie/gopm/internal/zap/encoder"
	"github.com/stuartcarnie/gopm/pkg/env"
	"github.com/stuartcarnie/gopm/process"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	cfg := zap.NewDevelopmentConfig()
	encoding := "term-color"
	if os.Getenv("NO_COLOR") != "" {
		encoding = "term"
	}
	cfg.Encoding = encoding
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

	kvs, err := env.ReadFile(rootOpt.EnvFile)
	if err != nil {
		zap.L().Error("Failed to open environment file", zap.String("file", rootOpt.EnvFile))
		return
	}
	for i := range kvs {
		kv := &kvs[i]
		err = os.Setenv(kv.Key, kv.Value)
		if err != nil {
			zap.L().Error("Failed to set environment variable", zap.String("key", kv.Key), zap.String("value", kv.Value), zap.Error(err))
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
