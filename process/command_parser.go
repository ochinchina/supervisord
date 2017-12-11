package process

import (
	"fmt"
	"unicode"
)

// find the position of byte ch in the string s start from offset
//
// return: -1 if byte ch is not found, >= offset if the ch is found
// in the string s from offset
func findChar(s string, offset int, ch byte) int {
	for i := offset; i < len(s); i++ {
		if s[i] == '\\' {
			i++
		} else if s[i] == ch {
			return i
		}
	}
	return -1
}

// skip all the white space and return the first position of non-space char
//
// return: the first position of non-space char or -1 if all the char
// from offset are space
func skipSpace(s string, offset int) int {
	for i := offset; i < len(s); i++ {
		if !unicode.IsSpace(rune(s[i])) {
			return i
		}
	}
	return -1
}

func appendArgument(arg string, args []string) []string {
	if arg[0] == '"' || arg[0] == '\'' {
		return append(args, arg[1:len(arg)-1])
	}
	return append(args, arg)
}

func parseCommand(command string) ([]string, error) {
	args := make([]string, 0)
	cmdLen := len(command)
	for i := 0; i < cmdLen; {
		//find the first non-space char
		j := skipSpace(command, i)
		if j == -1 {
			break
		}
		i = j
		for ; j < cmdLen; j++ {
			if unicode.IsSpace(rune(command[j])) {
				args = appendArgument(command[i:j], args)
				i = j + 1
				break
			} else if command[j] == '\\' {
				j++
			} else if command[j] == '"' || command[j] == '\'' {
				k := findChar(command, j+1, command[j])
				if k == -1 {
					args = appendArgument(command[i:], args)
					i = cmdLen
				} else {
					args = appendArgument(command[i:k+1], args)
					i = k + 1
				}
				break
			}
		}
		if j >= cmdLen {
			args = appendArgument(command[i:], args)
			i = cmdLen
		}
	}
	if len(args) <= 0 {
		return nil, fmt.Errorf("no command from empty string")
	}
	return args, nil
}
