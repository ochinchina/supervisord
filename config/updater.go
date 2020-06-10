package config

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/r3labs/diff"
	"github.com/stuartcarnie/gopm/model"
)

func ApplyUpdates(txn *memdb.Txn, m *model.Root) error {
	var u updater
	return u.update(txn, m)
}

type updater struct{}

func (u *updater) update(txn *memdb.Txn, m *model.Root) error {
	u.applyGroup(txn, m)
	u.applyHttpServer(txn, m)
	u.applyGrpcServer(txn, m)
	u.applyFileSystem(txn, m)
	u.applyPrograms(txn, m)
	return nil
}

func (u *updater) applyGroup(txn *memdb.Txn, m *model.Root) {
	for _, g := range m.Groups {
		obj := &Group{
			Name:     g.Name,
			Programs: g.Programs,
		}
		raw, _ := txn.First("group", "id", g.Name)
		if orig, ok := raw.(*Group); ok && !diff.Changed(orig, obj) {
			continue
		}
		_ = txn.Insert("group", obj)
	}
}

func (u *updater) applyPrograms(txn *memdb.Txn, m *model.Root) {
	for _, program := range m.Programs {
		proc := new(Process)
		proc.Name = program.Name

		if as := program.AutoRestart; as != nil {
			if *as {
				proc.AutoRestart = AutoStartModeAlways
			} else {
				proc.AutoRestart = AutoStartModeNever
			}
		} else {
			proc.AutoRestart = AutoStartModeDefault
		}

		proc.Group = program.Name // TODO(sgc): Add back groups
		proc.Name = program.Name
		proc.Directory = program.Directory
		proc.Command = program.Command
		proc.Environment = program.Environment
		proc.User = program.User
		proc.ExitCodes = program.ExitCodes
		proc.Priority = program.Priority
		proc.RestartPause = time.Duration(program.RestartPause)
		proc.StartRetries = program.StartRetries
		proc.StartSeconds = time.Duration(program.StartSeconds)
		proc.Cron = program.Cron
		proc.AutoStart = program.AutoStart
		proc.RestartDirectoryMonitor = program.RestartDirectoryMonitor
		proc.RestartFilePattern = program.RestartFilePattern
		proc.RestartWhenBinaryChanged = program.RestartWhenBinaryChanged
		proc.StopSignals = program.StopSignals
		proc.StopWaitSeconds = time.Duration(program.StopWaitSeconds)
		proc.StopAsGroup = program.StopAsGroup
		proc.KillAsGroup = program.KillAsGroup
		proc.StdoutLogFile = program.StdoutLogFile
		proc.StdoutLogfileBackups = program.StdoutLogfileBackups
		proc.StdoutLogFileMaxBytes = program.StdoutLogFileMaxBytes
		proc.RedirectStderr = program.RedirectStderr
		proc.StderrLogFile = program.StderrLogFile
		proc.StderrLogfileBackups = program.StderrLogfileBackups
		proc.StderrLogFileMaxBytes = program.StderrLogFileMaxBytes
		proc.DependsOn = program.DependsOn

		raw, _ := txn.First("process", "id", program.Name)
		if orig, ok := raw.(*Process); ok && !diff.Changed(orig, proc) {
			continue
		}
		if err := txn.Insert("process", proc); err != nil {
			panic(err)
		}
	}
}

func (u *updater) applyHttpServer(txn *memdb.Txn, m *model.Root) {
	if m.HttpServer == nil {
		_ = txn.Delete("server", &Server{Name: "http"})
		return
	}

	server := &Server{
		Name:     "http",
		Address:  m.HttpServer.Port,
		Username: m.HttpServer.Username,
		Password: m.HttpServer.Password,
	}

	raw, _ := txn.First("server", "id", "http")
	if orig, ok := raw.(*Server); ok && !diff.Changed(orig, server) {
		return
	}
	_ = txn.Insert("server", server)
}

func (u *updater) applyGrpcServer(txn *memdb.Txn, m *model.Root) {
	if m.GrpcServer == nil {
		_ = txn.Delete("server", &Server{Name: "grpc"})
		return
	}

	server := &Server{
		Name:     "grpc",
		Address:  m.GrpcServer.Address,
		Username: m.GrpcServer.Username,
		Password: m.GrpcServer.Password,
	}

	raw, _ := txn.First("server", "id", "grpc")
	if orig, ok := raw.(*Server); ok && !diff.Changed(orig, server) {
		return
	}
	_ = txn.Insert("server", server)
}

func (u *updater) applyFileSystem(txn *memdb.Txn, m *model.Root) {
	if m.FileSystem == nil {
		_, _ = txn.DeleteAll("file", "id", "")
		return
	}

	root := m.FileSystem.Root
	for _, mf := range m.FileSystem.Files {
		f := &File{
			Root:    root,
			Name:    mf.Name,
			Path:    mf.Path,
			Content: mf.Content,
		}

		raw, _ := txn.First("file", "id", mf.Name)
		if orig, ok := raw.(*File); ok && !diff.Changed(orig, f) {
			return
		}
		_ = txn.Insert("file", f)
	}
}
