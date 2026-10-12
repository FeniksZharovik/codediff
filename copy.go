package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type copyMode int

const (
	modeNew copyMode = iota
	modeOld
	modeBoth
	modeDiff
)

var modeNames = []string{"Kode baru", "Kode lama", "Lama + baru", "Diff"}

type copyResult struct {
	Text    string
	Files   int
	Skipped []string
}

func isBinary(b []byte) bool {
	n := imin(len(b), 8000)
	return bytes.IndexByte(b[:n], 0) >= 0
}

func buildCopy(r *Repo, c Commit, files []File, m copyMode) copyResult {
	var sb strings.Builder
	res := copyResult{}
	section := func(title, body string) {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("===== " + title + " =====\n")
		sb.WriteString(strings.TrimRight(body, "\r\n"))
		sb.WriteString("\n")
	}
	parentShort := ""
	if len(c.Parent) >= 7 {
		parentShort = c.Parent[:7]
	}
	read := func(rev, path string) (string, error) {
		b, err := r.show(rev, path)
		if err != nil {
			return "", err
		}
		if isBinary(b) {
			return "", errors.New("biner")
		}
		return strings.ToValidUTF8(string(b), "\uFFFD"), nil
	}
	for _, f := range files {
		if f.Binary {
			res.Skipped = append(res.Skipped, f.Path+" (biner)")
			continue
		}
		added := false
		switch m {
		case modeDiff:
			d, err := r.diff(c, f.Path)
			if err != nil || strings.TrimSpace(d) == "" {
				res.Skipped = append(res.Skipped, f.Path+" (diff kosong)")
				break
			}
			section(fmt.Sprintf("%s  ·  diff  ·  %s", f.Path, c.Short), strings.ToValidUTF8(d, "\uFFFD"))
			added = true
		default:
			if m == modeOld || m == modeBoth {
				if f.Status != "A" && c.Parent != "" {
					if body, err := read(c.Parent, f.Path); err == nil {
						section(fmt.Sprintf("%s  ·  lama  ·  %s", f.Path, parentShort), body)
						added = true
					}
				}
			}
			if m == modeNew || m == modeBoth {
				if f.Status != "D" {
					if body, err := read(c.Hash, f.Path); err == nil {
						section(fmt.Sprintf("%s  ·  baru  ·  %s", f.Path, c.Short), body)
						added = true
					}
				}
			}
			if !added {
				reason := "tidak tersedia"
				if m == modeNew && f.Status == "D" {
					reason = "dihapus"
				}
				if m == modeOld && f.Status == "A" {
					reason = "file baru"
				}
				res.Skipped = append(res.Skipped, f.Path+" ("+reason+")")
			}
		}
		if added {
			res.Files++
		}
	}
	res.Text = sb.String()
	return res
}

func writeClipboard(text string) error {
	return platformClipboard(text)
}