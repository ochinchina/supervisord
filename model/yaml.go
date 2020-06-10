package model

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/goccy/go-yaml"
)

type Reader struct{}

func (r *Reader) LoadString(s string) (*Root, error) {
	return r.LoadReader(strings.NewReader(s))
}

func (r *Reader) MustLoadString(s string) *Root {
	m, err := r.LoadReader(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return m
}

func MustLoadString(s string) *Root {
	var r Reader
	return r.MustLoadString(s)
}

func (r *Reader) LoadReader(reader io.Reader) (*Root, error) {
	dec := yaml.NewDecoder(reader)
	var v Root
	err := dec.Decode(&v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Reader) LoadPath(path string) (*Root, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return r.LoadReader(bytes.NewReader(d))
}
