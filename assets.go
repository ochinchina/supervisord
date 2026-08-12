package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed webgui
var content embed.FS

var HTTP http.FileSystem

func init() {
	// Priority: STATIC_DIR > STATIC_ZIP > STATIC_EMBED > DEV_DIR
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		HTTP = http.Dir(dir)
		return
	}

	if zipPath := os.Getenv("STATIC_ZIP"); zipPath != "" {
		f, err := os.Open(zipPath)
		if err != nil {
			panic(err)
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			panic(err)
		}
		zr, err := zip.NewReader(f, fi.Size())
		if err != nil {
			f.Close()
			panic(err)
		}
		zs := newZipFS(zr)
		HTTP = zs
		return
	}

	// try embedded assets
	webgui, err := fs.Sub(content, "webgui")
	if err == nil {
		HTTP = http.FS(webgui)
		return
	}

	// fallback to development directory
	HTTP = http.Dir("./webgui")
}

type zipFS struct {
    files map[string]*zip.File
    dirs  map[string][]string
}

func newZipFS(r *zip.Reader) *zipFS {
    z := &zipFS{files: make(map[string]*zip.File), dirs: make(map[string][]string)}
    for _, f := range r.File {
        name := path.Clean(f.Name)
        name = strings.TrimPrefix(name, "./")
        name = strings.TrimPrefix(name, "/")
        if f.FileInfo().IsDir() {
            // ensure directory key ends with /
            dir := strings.TrimSuffix(name, "/") + "/"
            z.dirs[dir] = []string{}
            continue
        }
        z.files[name] = f
        // populate parent directories
        dir := path.Dir(name)
        if dir == "." {
            dir = ""
        }
        for {
            key := dir
            if key != "" {
                key = strings.TrimPrefix(key, "/")
                key = strings.TrimSuffix(key, "/") + "/"
            } else {
                key = ""
            }
            z.dirs[key] = append(z.dirs[key], path.Base(name))
            if dir == "" || dir == "." || dir == "/" {
                break
            }
            dir = path.Dir(dir)
            if dir == "." {
                dir = ""
            }
        }
    }
    return z
}

func (z *zipFS) Open(name string) (http.File, error) {
    // normalize
    name = path.Clean(name)
    name = strings.TrimPrefix(name, "/")
    if name == "." || name == "" {
        // root directory
        return &zipHTTPFile{isDir: true, name: "", entries: z.dirEntries("")}, nil
    }

    // directory
    if strings.HasSuffix(name, "/") {
        dirKey := strings.TrimSuffix(name, "/") + "/"
        return &zipHTTPFile{isDir: true, name: dirKey, entries: z.dirEntries(dirKey)}, nil
    }

    // file
    if f, ok := z.files[name]; ok {
        rc, err := f.Open()
        if err != nil {
            return nil, err
        }
        // read full content into memory so we can provide Seek
        buf := new(bytes.Buffer)
        _, err = io.Copy(buf, rc)
        rc.Close()
        if err != nil {
            return nil, err
        }
        return &zipHTTPFile{isDir: false, name: path.Base(name), data: buf.Bytes(), fi: f.FileHeader.FileInfo()}, nil
    }

    // maybe the name refers to a directory without trailing slash
    dirKey := strings.TrimSuffix(name, "/") + "/"
    if _, ok := z.dirs[dirKey]; ok {
        return &zipHTTPFile{isDir: true, name: dirKey, entries: z.dirEntries(dirKey)}, nil
    }

    return nil, os.ErrNotExist
}

func (z *zipFS) dirEntries(dirKey string) []os.FileInfo {
    res := []os.FileInfo{}
    names, ok := z.dirs[dirKey]
    if !ok {
        return res
    }
    for _, n := range names {
        // try file
        full := strings.TrimPrefix(path.Join(strings.TrimSuffix(dirKey, "/"), n), "/")
        if f, ok := z.files[full]; ok {
            res = append(res, f.FileHeader.FileInfo())
            continue
        }
        // directory
        di := &zipFileInfo{name: n, isDir: true, modTime: time.Time{}, size: 0}
        res = append(res, di)
    }
    return res
}

type zipHTTPFile struct {
    isDir  bool
    name   string
    data   []byte
    rdr    *bytes.Reader
    fi     os.FileInfo
    entries []os.FileInfo
}

func (f *zipHTTPFile) Close() error { return nil }

func (f *zipHTTPFile) Read(p []byte) (int, error) {
    if f.isDir {
        return 0, io.EOF
    }
    if f.rdr == nil {
        f.rdr = bytes.NewReader(f.data)
    }
    return f.rdr.Read(p)
}

func (f *zipHTTPFile) Seek(offset int64, whence int) (int64, error) {
    if f.isDir {
        return 0, io.EOF
    }
    if f.rdr == nil {
        f.rdr = bytes.NewReader(f.data)
    }
    return f.rdr.Seek(offset, whence)
}

func (f *zipHTTPFile) Readdir(count int) ([]os.FileInfo, error) {
    if !f.isDir {
        return nil, os.ErrInvalid
    }
    if f.entries == nil {
        return nil, nil
    }
    if count <= 0 || count >= len(f.entries) {
        return f.entries, nil
    }
    res := f.entries[:count]
    f.entries = f.entries[count:]
    return res, nil
}

func (f *zipHTTPFile) Stat() (os.FileInfo, error) {
    if f.isDir {
        return &zipFileInfo{name: path.Base(strings.TrimSuffix(f.name, "/")), isDir: true, modTime: time.Time{}, size: 0}, nil
    }
    if f.fi != nil {
        return f.fi, nil
    }
    return &zipFileInfo{name: f.name, isDir: false, modTime: time.Time{}, size: int64(len(f.data))}, nil
}

type zipFileInfo struct {
    name    string
    isDir   bool
    modTime time.Time
    size    int64
}

func (z *zipFileInfo) Name() string       { return z.name }
func (z *zipFileInfo) Size() int64        { return z.size }
func (z *zipFileInfo) Mode() os.FileMode  { if z.isDir { return os.ModeDir | 0555 } ; return 0444 }
func (z *zipFileInfo) ModTime() time.Time { return z.modTime }
func (z *zipFileInfo) IsDir() bool        { return z.isDir }
func (z *zipFileInfo) Sys() interface{}   { return nil }
