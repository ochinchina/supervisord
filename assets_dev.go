// +build !release
//go:generate go run github.com/UnnoTed/fileb0x b0x.yaml

package main

import (
	"net/http"
)

var HTTP http.FileSystem = http.Dir("./webgui")
