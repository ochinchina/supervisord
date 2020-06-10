package env

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"unicode"
)

type KeyValue struct {
	Key   string
	Value string
}

type KeyValues []KeyValue

func Read(r io.Reader) (KeyValues, error) {
	reader := bufio.NewReader(r)
	var kvs KeyValues
	for {
		// for each line
		line, err := reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				break
			}
			// if the last line does not have a newline
			// still process it
			if len(line) == 0 {
				break
			}
		}
		// if line starts with '#', it is a comment line, ignore it
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] == '#' {
			continue
		}
		// if environment variable is exported with "export"
		if strings.HasPrefix(line, "export") && len(line) > len("export") && unicode.IsSpace(rune(line[len("export")])) {
			line = strings.TrimSpace(line[len("export"):])
		}
		// split the environment variable with "="
		pos := strings.Index(line, "=")
		if pos != -1 {
			k := strings.TrimSpace(line[0:pos])
			v := strings.TrimSpace(line[pos+1:])
			// if key and value are not empty, put it into the environment
			if len(k) > 0 && len(v) > 0 {
				kvs = append(kvs, KeyValue{Key: k, Value: v})
			}
		}
	}

	return kvs, nil
}

func ReadFile(name string) (KeyValues, error) {
	// try to open the rootOpt file
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Read(bufio.NewReader(f))
}
