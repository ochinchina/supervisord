package config

import (
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/stuartcarnie/gopm/model"
	"go.uber.org/multierr"
)

type environmentExpansion struct {
	err error
}

func (e *environmentExpansion) Err() error { return e.err }

func (e *environmentExpansion) Visit(node model.Node) model.Visitor {
	switch n := node.(type) {
	case *model.Program:
		n.Directory = e.expand(n.Directory)

	case *model.FileSystem:
		n.Root = e.expand(n.Root)

	case *model.File:
		n.Path = e.expand(n.Path)
	}

	return e
}

func (e *environmentExpansion) expand(path string) (newpath string) {
	var err error
	newpath, err = homedir.Expand(path)
	if err != nil {
		multierr.AppendInto(&e.err, err)
		return path
	}
	newpath = os.ExpandEnv(newpath)
	return newpath
}

func ExpandEnv(m *model.Root) error {
	var v environmentExpansion
	model.Walk(&v, m)
	return v.Err()
}
