//go:build !release
// +build !release

package main

import (
	"net/http"
)

//HTTP auto generated
var HTTP http.FileSystem = http.Dir("./webgui")
