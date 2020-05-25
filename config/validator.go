package config

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/stuartcarnie/gopm/model"
)

type validator struct {
	errors ErrList
}

func (v *validator) Err() error {
	return v.errors.Err()
}

func (v *validator) Visit(node model.Node) model.Visitor {
	switch n := node.(type) {
	case *model.HTTPServer:

	case *model.Program:
		if len(n.Name) == 0 {
			v.errors.Add(fmt.Errorf("missing program name"))
			// won't process anymore, as we have no name
			return v
		}

		if len(n.Cron) > 0 {
			p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			_, err := p.Parse(n.Cron)
			if err != nil {
				v.errors.Add(fmt.Errorf("error parsing cron for program %q, cron %q: %w", n.Name, n.Cron, err))
			}
		}

		if len(n.Command) == 0 {
			v.errors.Add(fmt.Errorf("missing command for program %q", n.Name))
		}
	}

	return v
}

func Validate(m *model.Root) error {
	var v validator
	model.Walk(&v, m)
	return v.Err()
}
