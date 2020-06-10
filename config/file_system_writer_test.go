package config_test

import (
	"io"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/mocks"
)

func TestFileSystemWriter(t *testing.T) {
	t.Run("FileSystem", func(t *testing.T) {
		t.Run("will create missing root dir", func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()

			fs := mocks.NewMockFs(ctl)

			rootFi := mocks.NewMockFileInfo(ctl)
			rootFi.EXPECT().
				IsDir().
				Return(false)

			fs.EXPECT().
				Stat("/etc").
				Return(rootFi, nil)
			fs.EXPECT().
				MkdirAll("/etc", os.ModePerm).
				Return(nil)

			fsw := config.NewFileSystemWriter(fs)
			_, _, err := fsw.Commit("/etc", nil)
			assert.NoError(t, err)
		})

		t.Run("will skip existing root dir", func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()

			fs := mocks.NewMockFs(ctl)

			rootFi := mocks.NewMockFileInfo(ctl)
			rootFi.EXPECT().
				IsDir().
				Return(true)

			fs.EXPECT().
				Stat("/etc").
				Return(rootFi, nil)

			fsw := config.NewFileSystemWriter(fs)
			_, _, err := fsw.Commit("/etc", nil)
			assert.NoError(t, err)
		})
	})

	t.Run("File", func(t *testing.T) {
		t.Run("does not write matching content", func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()

			fs := mocks.NewMockFs(ctl)

			rootFi := mocks.NewMockFileInfo(ctl)
			rootFi.EXPECT().
				IsDir().
				Return(true)

			fs.EXPECT().
				Stat("/etc").
				Return(rootFi, nil)

			fileDirFi := mocks.NewMockFileInfo(ctl)
			fileDirFi.EXPECT().
				IsDir().
				Return(false)

			fs.EXPECT().
				Stat("/etc/caddy").
				Return(fileDirFi, nil)
			fs.EXPECT().MkdirAll("/etc/caddy", os.ModePerm).
				Return(nil)

			fileF := mocks.NewMockFile(ctl)
			fileF.EXPECT().
				Read(gomock.Any()).
				DoAndReturn(func(d []byte) (int, error) {
					copy(d, "foo")
					return 3, io.EOF
				})

			fs.EXPECT().
				Stat("/etc/caddy/Caddyfile").
				Return(nil, nil)
			fs.EXPECT().
				Open("/etc/caddy/Caddyfile").
				Return(fileF, nil)

			hash := mocks.NewMockHash(ctl)
			hash.EXPECT().
				Reset().
				Times(2)
			hash.EXPECT().
				Write(gomock.Any()).
				Times(2).
				Return(3, nil)
			hash.EXPECT().
				Sum(gomock.Any()).
				Times(2).
				Return([]byte("myhash"))

			files := []*config.File{{Path: "caddy/Caddyfile", Content: "hello"}}
			fsw := config.NewFileSystemWriter(fs, config.WithHasher(hash))
			_, _, err := fsw.Commit("/etc", files)
			assert.NoError(t, err)
		})

		t.Run("writes mismatched content", func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()

			fs := mocks.NewMockFs(ctl)

			rootFi := mocks.NewMockFileInfo(ctl)
			rootFi.EXPECT().
				IsDir().
				Return(true)

			fs.EXPECT().
				Stat("/etc").
				Return(rootFi, nil)

			fileDirFi := mocks.NewMockFileInfo(ctl)
			fileDirFi.EXPECT().
				IsDir().
				Return(false)

			fs.EXPECT().
				Stat("/etc/caddy").
				Return(fileDirFi, nil)
			fs.EXPECT().
				MkdirAll("/etc/caddy", os.ModePerm).
				Return(nil)

			fileF := mocks.NewMockFile(ctl)
			fileF.EXPECT().
				Read(gomock.Any()).
				DoAndReturn(func(d []byte) (int, error) {
					copy(d, "foo")
					return 3, io.EOF
				})

			fs.EXPECT().
				Stat("/etc/caddy/Caddyfile").
				Return(nil, nil)
			fs.EXPECT().
				Open("/etc/caddy/Caddyfile").
				Return(fileF, nil)

			hash := mocks.NewMockHash(ctl)
			hash.EXPECT().
				Reset().
				Times(2)
			hash.EXPECT().
				Write(gomock.Any()).
				Times(2).
				Return(3, nil)
			hash.EXPECT().
				Sum(gomock.Any()).
				Times(1).
				Return([]byte("myhash"))
			hash.EXPECT().
				Sum(gomock.Any()).
				Times(1).
				Return([]byte("my-new-hash"))

			fileF2 := mocks.NewMockFile(ctl)
			fileF2.EXPECT().
				Write(gomock.Any()).
				DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
			fileF2.EXPECT().Close().Return(nil)

			fs.EXPECT().
				OpenFile("/etc/caddy/Caddyfile", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm).
				Return(fileF2, nil)

			files := []*config.File{{Path: "caddy/Caddyfile", Content: "hello"}}
			fsw := config.NewFileSystemWriter(fs, config.WithHasher(hash))
			_, lf, err := fsw.Commit("/etc", files)
			assert.NoError(t, err)
			assert.Len(t, lf, 1)
		})
	})
}
