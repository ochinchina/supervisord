// +build !release

package gopm

import (
	"net/http"
)

// HTTP auto generated
var HTTP http.FileSystem = http.Dir("./webgui")
