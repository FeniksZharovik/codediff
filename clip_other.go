//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func platformClipboard(text string) error {
	if runtime.GOOS == "darwin" {
		return pipeTo(text, "pbcopy")
	}
	candidates := [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err == nil {
			return pipeTo(text, c[0], c[1:]...)
		}
	}
	return errors.New("utilitas clipboard tidak ditemukan (pasang wl-clipboard, xclip, atau xsel)")
}

func pipeTo(text, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}