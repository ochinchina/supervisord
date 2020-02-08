// +build !release
//go:generate go run github.com/UnnoTed/fileb0x b0x.yaml

package main

import (
	"net/http"
)

// HTTP is link to webgui directory
var HTTP http.FileSystem = http.Dir("./webgui")
