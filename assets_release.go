// +build release
//go:generate go run github.com/rakyll/statik -src=./webgui -p=assets_generated

package gopm

import (
	"net/http"

	"github.com/rakyll/statik/fs"
	_ "github.com/stuartcarnie/gopm/assets_generated"
	"go.uber.org/zap"
)

// HTTP auto generated
var HTTP http.FileSystem = func() http.FileSystem {
	statikFS, err := fs.New()
	if err != nil {
		zap.L().Fatal("Could not load embedded assets.", zap.Error(err))
	}
	return statikFS
}()
