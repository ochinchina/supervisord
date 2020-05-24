package yaml

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/goccy/go-yaml"
	"github.com/stuartcarnie/gopm/model"
)

type Reader struct{}

func (r *Reader) LoadReader(reader io.Reader) (*model.Root, error) {
	dec := yaml.NewDecoder(reader)
	var v model.Root
	err := dec.Decode(&v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Reader) LoadPath(path string) (*model.Root, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return r.LoadReader(bytes.NewReader(d))
}
