package model

import (
	"errors"
	"fmt"

	"github.com/robfig/cron/v3"
	"go.uber.org/multierr"
)

type validator struct {
	err error
}

func (v *validator) Err() error {
	return v.err
}

func (v *validator) Visit(node Node) Visitor {
	switch n := node.(type) {
	case *HTTPServer:

	case *Program:
		if len(n.Name) == 0 {
			multierr.AppendInto(&v.err, errors.New("program name missing"))
			// won't process anymore, as we have no name
			return v
		}

		if len(n.Cron) > 0 {
			p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			_, err := p.Parse(n.Cron)
			if err != nil {
				multierr.AppendInto(&v.err, fmt.Errorf("error parsing cron for program %q, cron %q: %w", n.Name, n.Cron, err))
			}
		}

		if len(n.Command) == 0 {
			multierr.AppendInto(&v.err, fmt.Errorf("missing command for program %q", n.Name))
		}

	case *FileSystem:
		if len(n.Root) == 0 {
			multierr.AppendInto(&v.err, errors.New("filesystem.root is required"))
		}

		for _, f := range n.Files {
			if len(f.Name) == 0 {
				multierr.AppendInto(&v.err, errors.New("file name missing"))
				continue
			}

			if len(f.Path) == 0 {
				multierr.AppendInto(&v.err, fmt.Errorf("path missing for file: %s", f.Name))
			}

			if len(f.Content) == 0 {
				multierr.AppendInto(&v.err, fmt.Errorf("content missing for file: %s", f.Name))
			}
		}
	}

	return v
}

func Validate(m *Root) error {
	var v validator
	Walk(&v, m)
	return v.Err()
}
