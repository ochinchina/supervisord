package process

import (
	"os/user"
	"path/filepath"
)

func path_split(path string) []string {
	r := make([]string, 0)
	cur_path := path
	for {
		dir, file := filepath.Split(cur_path)
		if len(file) > 0 {
			r = append(r, file)
		}
		if len(dir) <= 0 {
			break
		}
		cur_path = dir[0 : len(dir)-1]
	}
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}
func Path_expand(path string) (string, error) {
	pathList := path_split(path)

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
