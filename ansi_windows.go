//go:build windows

package main

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	kern       = syscall.NewLazyDLL("kernel32.dll")
	getMode    = kern.NewProc("GetConsoleMode")
	setMode    = kern.NewProc("SetConsoleMode")
	setOutCP   = kern.NewProc("SetConsoleOutputCP")
	setInCP    = kern.NewProc("SetConsoleCP")
	getBufInfo = kern.NewProc("GetConsoleScreenBufferInfo")
)

func enableANSI() {
	setOutCP.Call(65001)
	setInCP.Call(65001)
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		var m uint32
		if r, _, _ := getMode.Call(f.Fd(), uintptr(unsafe.Pointer(&m))); r != 0 {
			setMode.Call(f.Fd(), uintptr(m|0x0004))
		}
	}
}

func makeRaw() (func(), error) {
	h := os.Stdin.Fd()
	var m uint32
	if r, _, _ := getMode.Call(h, uintptr(unsafe.Pointer(&m))); r == 0 {
		return nil, errors.New("input bukan terminal interaktif")
	}
	n := (m &^ (0x0001 | 0x0002 | 0x0004)) | 0x0200
	if r, _, _ := setMode.Call(h, uintptr(n)); r == 0 {
		return nil, errors.New("gagal mengatur mode terminal")
	}
	return func() { setMode.Call(h, uintptr(m)) }, nil
}

func termSize() (int, int) {
	var info struct {
		size    [2]int16
		cursor  [2]int16
		attrs   uint16
		window  [4]int16
		maxSize [2]int16
	}
	r, _, _ := getBufInfo.Call(os.Stdout.Fd(), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 100, 30
	}
	w := int(info.window[2]-info.window[0]) + 1
	h := int(info.window[3]-info.window[1]) + 1
	if w < 20 || h < 5 {
		return 100, 30
	}
	return w, h
}