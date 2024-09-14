//go:build release
// +build release

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed webgui
var content embed.FS

var HTTP http.FileSystem

func init() {
	webgui, err := fs.Sub(content, "webgui")
	if err != nil {
		panic(err)
	}

	HTTP = http.FS(webgui)
}
