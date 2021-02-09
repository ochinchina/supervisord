// +build windows

package main

func Deamonize(logfile string, proc func()) {
	proc()
}
