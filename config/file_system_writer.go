package config

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type FileSystemWriter struct {
	RootDir string
	Files   []*File
	fs      afero.Afero
	hash    hash.Hash
}

type OptionFn func(v *FileSystemWriter)

func WithHasher(h hash.Hash) OptionFn {
	return func(v *FileSystemWriter) {
		v.hash = h
	}
}

func NewFileSystemWriter(fs afero.Fs, opts ...OptionFn) *FileSystemWriter {
	v := &FileSystemWriter{
		fs: afero.Afero{Fs: fs},
	}

	for _, opt := range opts {
		opt(v)
	}

	if v.hash == nil {
		v.hash = sha256.New()
	}

	return v
}

func (v *FileSystemWriter) Commit(root string, files []*File) (string, []*LocalFile, error) {
	dir, err := v.checkRoot(root)
	if err != nil {
		return "", nil, err
	}

	if len(files) == 0 {
		return "", nil, nil
	}

	fs := afero.Afero{Fs: afero.NewBasePathFs(v.fs.Fs, dir)}

	localFiles := make([]*LocalFile, 0, len(files))
	for _, of := range files {
		lf, err := v.checkFile(fs, of)
		if err != nil {
			return "", nil, err
		}
		localFiles = append(localFiles, lf)
	}

	return dir, localFiles, nil
}

func (v *FileSystemWriter) checkFile(fs afero.Afero, n *File) (*LocalFile, error) {
	if filepath.IsAbs(n.Path) {
		return nil, fmt.Errorf("file.path error: %q file must be relative path: %q", n.Name, n.Path)
	}

	// check / create dir
	dir := filepath.Dir(n.Path)
	if ok, err := fs.DirExists(dir); err != nil {
		return nil, fmt.Errorf("DirExists error: %q: %w", dir, err)
	} else if !ok {
		if err := fs.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("file.path error: unable to create dir %q: %w", dir, err)
		}
	}

	// check / create file
	var dstHash []byte
	if ok, err := fs.Exists(n.Path); err != nil {
		return nil, fmt.Errorf("exists error: %q: %w", n.Path, err)
	} else if ok {
		v.hash.Reset()
		if file, err := fs.Open(n.Path); err == nil {
			_, _ = io.Copy(v.hash, file)
		}
		dstHash = v.hash.Sum(nil)
	}

	fullPath, err := fs.Fs.(*afero.BasePathFs).RealPath(n.Path)
	if err != nil {
		return nil, fmt.Errorf("RealPath error: %q: %w", n.Path, err)
	}

	lf := &LocalFile{Name: n.Name, FullPath: fullPath}

	v.hash.Reset()
	_, _ = io.Copy(v.hash, strings.NewReader(n.Content))
	lf.Hash = v.hash.Sum(nil)

	if !bytes.Equal(lf.Hash, dstHash) {
		if err := fs.WriteFile(n.Path, []byte(n.Content), os.ModePerm); err != nil {
			return nil, fmt.Errorf("file write: unable to write file %q: %w", n.Path, err)
		}
	}

	return lf, nil
}

func (v *FileSystemWriter) checkRoot(dir string) (string, error) {
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	if ok, err := v.fs.DirExists(dir); err != nil {
		return "", err
	} else if !ok {
		if err := v.fs.MkdirAll(dir, os.ModePerm); err != nil {
			return "", err
		}
	}
	return dir, nil
}
