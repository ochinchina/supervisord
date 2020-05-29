package config

import (
	"os"

	"github.com/stuartcarnie/gopm/model"
)

type environmentExpansion struct{}

func (e environmentExpansion) Visit(node model.Node) model.Visitor {
	switch n := node.(type) {
	case *model.Program:
		n.Directory = os.ExpandEnv(n.Directory)
	}

	return e
}

func ExpandEnv(m *model.Root) {
	var v environmentExpansion
	model.Walk(&v, m)
}
