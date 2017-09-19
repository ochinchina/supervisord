package main

// +build windows

func Deamonize(proc func()) {
	proc()
}
