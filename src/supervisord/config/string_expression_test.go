package config

import (
	"os"
	"testing"
)

func TestEval(t *testing.T) {
	se := NewStringExpression()

	se.Add("var1", "ok").Add("var2", "2")

	r, _ := se.Eval("%(var1)s_test_%(var2)02d")

	if r != "ok_test_02" {
		t.Error("fail to replace the environment")
	}
}

func TestEnv(t *testing.T) {
	os.Setenv("FOO", "BAR=BAZ")

	se := NewStringExpression()

	r, _ := se.Eval("%(ENV_FOO)s")

	if r != "BAR=BAZ" {
		t.Errorf("fail to replace the environment: %s", r)
	}
}
