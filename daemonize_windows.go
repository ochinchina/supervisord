// +build windows

package main

func Daemonize(logfile string, proc func()) {
	proc()
}
