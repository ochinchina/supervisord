package model

import "time"

type Duration time.Duration

func (d *Duration) UnmarshalYAML(bytes []byte) error {
	v, err := time.ParseDuration(string(bytes))
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

type ByteUnits int
