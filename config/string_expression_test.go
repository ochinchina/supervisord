package config

import (
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
