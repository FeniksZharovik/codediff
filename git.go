package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type File struct {
	Status   string
	Path     string
	Add, Del int
	Binary   bool
}

type Commit struct {
	Hash, Short, Author, Subject, Parent string
	When                                 time.Time
	Merge                                bool
	Files                                []File
	Add, Del                             int
}

type Repo struct {
	Source, Name, Dir, Branch string
	Local, Stale              bool
}

var (
	ghRe     = regexp.MustCompile(`^(?:(?:https?://)?(?:www\.)?|git@)github\.com[/:]([\w.-]+)/([\w.-]+?)(?:\.git)?(?:[/?#].*)?$`)
	shortRe  = regexp.MustCompile(`^([\w.-]+)/([\w.-]+?)(?:\.git)?$`)
	remoteRe = regexp.MustCompile(`^(?:https?|ssh|git)://`)
	progRe   = regexp.MustCompile(`(Enumerating objects|Counting objects|Compressing objects|Receiving objects|Resolving deltas):\s+(\d+)%`)
	safeRe   = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

func gitCmd(dir string, args ...string) *exec.Cmd {
	full := []string{"-c", "core.quotepath=false"}
	if dir != "" {
		full = append(full, "-C", dir)
	}
	full = append(full, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "LANGUAGE=C")
	return cmd
}

func gitOut(dir string, args ...string) ([]byte, error) {
	cmd := gitCmd(dir, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return out, nil
}

func gitVersion() (string, error) {
	out, err := gitOut("", "--version")
	if err != nil {
		return "", errors.New("git tidak ditemukan — pasang dari https://git-scm.com")
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "git version "), nil
}

func parseSource(in string) (string, string, bool, error) {
	in = strings.Trim(strings.TrimSpace(in), `"'`)
	if in == "" {
		return "", "", false, errors.New("URL repositori kosong")
	}
	if st, err := os.Stat(in); err == nil && st.IsDir() {
		abs, _ := filepath.Abs(in)
		if _, err := gitOut(abs, "rev-parse", "--git-dir"); err != nil {
			return "", "", false, fmt.Errorf("%s bukan repositori git", abs)
		}
		return abs, filepath.Base(abs), true, nil
	}
	if m := ghRe.FindStringSubmatch(in); m != nil {
		return "https://github.com/" + m[1] + "/" + m[2] + ".git", m[1] + "/" + m[2], false, nil
	}
	if remoteRe.MatchString(in) {
		parts := strings.Split(strings.TrimSuffix(strings.TrimSuffix(in, "/"), ".git"), "/")
		name := parts[len(parts)-1]
		if len(parts) >= 2 {
			name = parts[len(parts)-2] + "/" + name
		}
		return in, name, false, nil
	}
	if m := shortRe.FindStringSubmatch(in); m != nil {
		return "https://github.com/" + m[1] + "/" + m[2] + ".git", m[1] + "/" + m[2], false, nil
	}
	return "", "", false, errors.New("format tidak dikenali — gunakan https://github.com/pemilik/repo atau pemilik/repo")
}

func cacheDir(src string) string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	name := strings.Trim(safeRe.ReplaceAllString(src, "_"), "_")
	if len(name) > 100 {
		name = name[len(name)-100:]
	}
	return filepath.Join(base, "codediff", name)
}

func splitCR(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func stageInfo(stage string, n int) (string, float64) {
	f := float64(n)
	switch stage {
	case "Enumerating objects", "Counting objects":
		return "menghitung objek", f * 0.10
	case "Compressing objects":
		return "memadatkan objek", 10 + f*0.10
	case "Receiving objects":
		return "menerima objek", 20 + f*0.65
	}
	return "menyusun delta", 85 + f*0.15
}

func explainGit(tail []string) error {
	all := strings.ToLower(strings.Join(tail, " "))
	switch {
	case strings.Contains(all, "not found"), strings.Contains(all, "could not read username"),
		strings.Contains(all, "authentication failed"), strings.Contains(all, "terminal prompts disabled"):
		return errors.New("repositori tidak ditemukan atau bersifat privat")
	case strings.Contains(all, "could not resolve host"), strings.Contains(all, "unable to access"),
		strings.Contains(all, "failed to connect"):
		return errors.New("tidak dapat terhubung ke server — periksa koneksi internet")
	}
	if len(tail) > 0 {
		return errors.New(tail[len(tail)-1])
	}
	return errors.New("perintah git gagal")
}

func runProgress(cmd *exec.Cmd, p *prog) error {
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var tail []string
	sc := bufio.NewScanner(stderr)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	sc.Split(splitCR)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sc.Text()), "remote:"))
		if line == "" {
			continue
		}
		if m := progRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[2])
			label, pct := stageInfo(m[1], n)
			p.set(label, pct)
			continue
		}
		tail = append(tail, line)
		if len(tail) > 6 {
			tail = tail[1:]
		}
	}
	if err := cmd.Wait(); err != nil {
		return explainGit(tail)
	}
	return nil
}

func prepareRepo(src, name string, local bool, branch string, p *prog) (*Repo, error) {
	r := &Repo{Source: src, Name: name, Branch: branch, Local: local}
	if local {
		r.Dir = src
		return r, nil
	}
	dir := cacheDir(src)
	r.Dir = dir
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err == nil {
		p.set("memperbarui", 0)
		if err := runProgress(gitCmd(dir, "fetch", "--prune", "--progress", "origin", "+refs/heads/*:refs/heads/*"), p); err != nil {
			r.Stale = true
		}
		return r, nil
	}
	os.RemoveAll(dir)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, err
	}
	if err := runProgress(gitCmd("", "clone", "--bare", "--progress", src, dir), p); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return r, nil
}

func resolveBranch(r *Repo) error {
	if r.Branch != "" {
		if _, err := gitOut(r.Dir, "rev-parse", "--verify", "--quiet", r.Branch+"^{commit}"); err != nil {
			return fmt.Errorf("branch atau ref '%s' tidak ditemukan", r.Branch)
		}
		return nil
	}
	if out, err := gitOut(r.Dir, "symbolic-ref", "--short", "HEAD"); err == nil {
		r.Branch = strings.TrimSpace(string(out))
		return nil
	}
	r.Branch = "HEAD"
	return nil
}

func readLog(r *Repo, limit int, p *prog) ([]Commit, error) {
	args := []string{"log", "--no-renames", "--root", "--raw", "--numstat", "--no-color",
		"--pretty=format:%x1e%H%x1f%h%x1f%an%x1f%at%x1f%P%x1f%s"}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}
	args = append(args, r.Branch, "--")
	cmd := gitCmd(r.Dir, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	rd := bufio.NewReaderSize(stdout, 1<<20)
	var commits []Commit
	for {
		rec, rerr := rd.ReadString(0x1e)
		rec = strings.TrimSuffix(rec, "\x1e")
		if strings.TrimSpace(rec) != "" {
			if c, ok := parseRecord(rec); ok {
				commits = append(commits, c)
				if len(commits)%25 == 0 {
					p.set(num(len(commits))+" commit", -1)
				}
			}
		}
		if rerr != nil {
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "does not have any commits") {
			return nil, errors.New("repositori belum memiliki commit")
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	if len(commits) == 0 {
		return nil, errors.New("tidak ada commit yang ditemukan")
	}
	return commits, nil
}

func parseRecord(rec string) (Commit, bool) {
	head, body := rec, ""
	if nl := strings.IndexByte(rec, '\n'); nl >= 0 {
		head, body = rec[:nl], rec[nl+1:]
	}
	f := strings.SplitN(head, "\x1f", 6)
	if len(f) < 6 {
		return Commit{}, false
	}
	ts, _ := strconv.ParseInt(f[3], 10, 64)
	c := Commit{
		Hash:    f[0],
		Short:   f[1],
		Author:  strings.TrimSpace(f[2]),
		When:    time.Unix(ts, 0),
		Subject: strings.Join(strings.Fields(f[5]), " "),
	}
	if ps := strings.Fields(f[4]); len(ps) > 0 {
		c.Parent = ps[0]
		c.Merge = len(ps) > 1
	}
	status := map[string]string{}
	stats := map[string][2]int{}
	bin := map[string]bool{}
	var order []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimRight(l, "\r")
		if l == "" {
			continue
		}
		if l[0] == ':' {
			parts := strings.SplitN(l, "\t", 2)
			meta := strings.Fields(parts[0])
			if len(parts) == 2 && len(meta) >= 5 {
				if _, seen := status[parts[1]]; !seen {
					order = append(order, parts[1])
				}
				status[parts[1]] = meta[4][:1]
			}
			continue
		}
		parts := strings.SplitN(l, "\t", 3)
		if len(parts) == 3 {
			a, e1 := strconv.Atoi(parts[0])
			d, e2 := strconv.Atoi(parts[1])
			if e1 != nil || e2 != nil {
				bin[parts[2]] = true
			} else {
				stats[parts[2]] = [2]int{a, d}
			}
		}
	}
	sort.Strings(order)
	for _, p := range order {
		s := stats[p]
		c.Files = append(c.Files, File{Status: status[p], Path: p, Add: s[0], Del: s[1], Binary: bin[p]})
		c.Add += s[0]
		c.Del += s[1]
	}
	return c, true
}

func (r *Repo) show(rev, path string) ([]byte, error) {
	return gitOut(r.Dir, "show", rev+":"+path)
}

func (r *Repo) diff(c Commit, path string) (string, error) {
	out, err := gitOut(r.Dir, "show", "--no-color", "--no-renames", "--format=", c.Hash, "--", path)
	return string(out), err
}