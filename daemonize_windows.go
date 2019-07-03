// +build windows

package main

func Deamonize(proc func()) {
	proc()
}
