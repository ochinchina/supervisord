package config

import (
	"strings"
)

type ErrList struct {
	errs []error
}

func (e *ErrList) Add(err error) {
	if err == nil {
		return
	}
	e.errs = append(e.errs, err)
}

func (e *ErrList) Error() string {
	if len(e.errs) == 0 {
		return ""
	}

	if len(e.errs) == 1 {
		return e.errs[0].Error()
	}

	var b strings.Builder
	for _, err := range e.errs {
		b.WriteString(err.Error())
		b.WriteByte('\n')
	}
	return b.String()
}

func (e *ErrList) Err() error {
	if len(e.errs) == 0 {
		return nil
	}
	return e
}

func (e *ErrList) Errors() []error {
	return e.errs
}
