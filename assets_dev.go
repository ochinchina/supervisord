// +build !release
//go:generate go run github.com/UnnoTed/fileb0x b0x.yaml

package gopm

import (
	"net/http"
)

// HTTP auto generated
var HTTP http.FileSystem = http.Dir("./webgui")
