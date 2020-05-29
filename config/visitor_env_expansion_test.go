package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/model"
)

func TestExpandEnv(t *testing.T) {
	m := model.Root{
		Programs: []*model.Program{
			{Directory: "before/${TEST_VAR}/after"},
		},
	}

	val := "THIS IS A TEST"
	err := os.Setenv("TEST_VAR", val)
	assert.NoError(t, err)

	config.ExpandEnv(&m)
	exp := "before/" + val + "/after"
	assert.Equal(t, exp, m.Programs[0].Directory)
}
