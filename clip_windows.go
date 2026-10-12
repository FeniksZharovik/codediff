//go:build windows

package main

import (
	"errors"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	openClipboard  = user32.NewProc("OpenClipboard")
	closeClipboard = user32.NewProc("CloseClipboard")
	emptyClipboard = user32.NewProc("EmptyClipboard")
	setClipData    = user32.NewProc("SetClipboardData")
	globalAlloc    = kern.NewProc("GlobalAlloc")
	globalLock     = kern.NewProc("GlobalLock")
	globalUnlock   = kern.NewProc("GlobalUnlock")
	globalFree     = kern.NewProc("GlobalFree")
	moveMemory     = kern.NewProc("RtlMoveMemory")
)

func platformClipboard(text string) error {
	t := strings.ReplaceAll(text, "\x00", "")
	t = strings.ReplaceAll(strings.ReplaceAll(t, "\r\n", "\n"), "\n", "\r\n")
	u, err := syscall.UTF16FromString(t)
	if err != nil {
		return err
	}
	opened := false
	for i := 0; i < 10; i++ {
		if r, _, _ := openClipboard.Call(0); r != 0 {
			opened = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !opened {
		return errors.New("clipboard sedang dipakai aplikasi lain")
	}
	defer closeClipboard.Call()
	emptyClipboard.Call()
	h, _, _ := globalAlloc.Call(0x0002, uintptr(len(u)*2))
	if h == 0 {
		return errors.New("gagal mengalokasikan memori clipboard")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		globalFree.Call(h)
		return errors.New("gagal mengunci memori clipboard")
	}
	moveMemory.Call(p, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)*2))
	globalUnlock.Call(h)
	if r, _, _ := setClipData.Call(13, h); r == 0 {
		globalFree.Call(h)
		return errors.New("gagal menulis ke clipboard")
	}
	return nil
}