package process

import (
	"os/user"
	"path/filepath"
)

func pathSplit(path string) []string {
	r := make([]string, 0)
	curPath := path
	for {
		dir, file := filepath.Split(curPath)
		if len(file) > 0 {
			r = append(r, file)
		}
		if len(dir) <= 0 {
			break
		}
		curPath = dir[0 : len(dir)-1]
	}
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

// PathExpand replaces the ~ with user home directory
func PathExpand(path string) (string, error) {
	pathList := pathSplit(path)

	if len(pathList) > 0 && len(pathList[0]) > 0 && pathList[0][0] == '~' {
		var usr *user.User
		var err error

		if pathList[0] == "~" {
			usr, err = user.Current()
		} else {
			usr, err = user.Lookup(pathList[0][1:])
		}

		if err != nil {
			return "", err
		}
		pathList[0] = usr.HomeDir
		return filepath.Join(pathList...), nil
	}
	return path, nil
}
