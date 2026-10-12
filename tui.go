package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	kNone = iota
	kUp
	kDown
	kLeft
	kRight
	kPgUp
	kPgDn
	kHome
	kEnd
	kEnter
	kEsc
	kBack
	kSpace
	kTab
	kBackTab
	kCtrlC
	kRune
)

type key struct {
	k int
	r rune
}

var (
	pending []key
	leave   func()
)

func enterTUI() error {
	restore, err := makeRaw()
	if err != nil {
		return err
	}
	fmt.Print("\x1b[?1049h\x1b[?25l")
	leave = func() {
		fmt.Print("\x1b[?25h\x1b[?1049l")
		restore()
		leave = nil
	}
	return nil
}

func csiKey(seq string) int {
	switch seq {
	case "\x1b[A", "\x1bOA":
		return kUp
	case "\x1b[B", "\x1bOB":
		return kDown
	case "\x1b[C", "\x1bOC":
		return kRight
	case "\x1b[D", "\x1bOD":
		return kLeft
	case "\x1b[5~":
		return kPgUp
	case "\x1b[6~":
		return kPgDn
	case "\x1b[H", "\x1b[1~", "\x1bOH", "\x1b[7~":
		return kHome
	case "\x1b[F", "\x1b[4~", "\x1bOF", "\x1b[8~":
		return kEnd
	case "\x1b[Z":
		return kBackTab
	}
	return kNone
}

func parseKeys(b []byte) []key {
	var ks []key
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == 0x1b:
			if i+1 < len(b) && (b[i+1] == '[' || b[i+1] == 'O') {
				j := i + 2
				for j < len(b) && !(b[j] >= 0x40 && b[j] <= 0x7e) {
					j++
				}
				if j >= len(b) {
					return ks
				}
				if k := csiKey(string(b[i : j+1])); k != kNone {
					ks = append(ks, key{k: k})
				}
				i = j + 1
				continue
			}
			ks = append(ks, key{k: kEsc})
			i++
		case c == 3:
			ks = append(ks, key{k: kCtrlC})
			i++
		case c == '\r' || c == '\n':
			ks = append(ks, key{k: kEnter})
			i++
		case c == '\t':
			ks = append(ks, key{k: kTab})
			i++
		case c == ' ':
			ks = append(ks, key{k: kSpace})
			i++
		case c == 0x7f || c == 0x08:
			ks = append(ks, key{k: kBack})
			i++
		case c < 32:
			i++
		default:
			r, size := utf8.DecodeRune(b[i:])
			ks = append(ks, key{k: kRune, r: r})
			i += size
		}
	}
	return ks
}

func nextKey() key {
	for len(pending) == 0 {
		buf := make([]byte, 64)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return key{k: kCtrlC}
		}
		pending = parseKeys(buf[:n])
	}
	k := pending[0]
	pending = pending[1:]
	return k
}

func inner(w int) int { return imax(20, w-6) }

func draw(lines []string, w, h int) {
	var sb strings.Builder
	sb.WriteString("\x1b[H")
	for i := 0; i < h; i++ {
		if i < len(lines) {
			sb.WriteString(truncate(lines[i], w-1))
		}
		sb.WriteString(reset + "\x1b[K")
		if i < h-1 {
			sb.WriteString("\r\n")
		}
	}
	os.Stdout.WriteString(sb.String())
}

func bodyLine(s string, w int) string { return "   " + truncate(s, inner(w)) }
func ruleLine(w int) string { return " " + rule(w-3) }

func joinLR(left, right string, width int) string {
	gap := width - visLen(left) - visLen(right)
	if gap < 1 {
		return truncate(left, imax(1, width-visLen(right)-1)) + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func rowLine(body string, selected bool, w int) string {
	mk := " "
	if selected {
		mk = accent + g.mark + reset
	}
	line := " " + mk + " " + fit(body, inner(w)) + "  "
	if selected && useColor {
		return withBg(line, selBg)
	}
	return line
}

func clean(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			sb.WriteString("    ")
		case r < 32 || r == 0x7f:
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func titleLine(repo *Repo, trail, right string, w int) string {
	left := bold + "codediff" + reset + dim + " " + g.arrow + " " + reset + clean(repo.Name) + dim + " " + g.sep + " " + clean(repo.Branch) + reset
	if trail != "" {
		left += dim + " " + g.arrow + " " + reset + trail
	}
	return "   " + joinLR(left, dim+right+reset, inner(w))
}

func keysLine(pairs [][2]string, right string, w int) string {
	var parts []string
	for _, p := range pairs {
		parts = append(parts, keyHint(p[0], p[1]))
	}
	return "   " + joinLR(strings.Join(parts, "   "), dim+right+reset, inner(w))
}

type colLayout struct {
	subj                       int
	files, delta, when, author bool
}

func layoutCols(width int) colLayout {
	l := colLayout{files: true, delta: true, when: true, author: true}
	for {
		fixed := 9
		if l.files {
			fixed += 10
		}
		if l.delta {
			fixed += 15
		}
		if l.when {
			fixed += 15
		}
		if l.author {
			fixed += 16
		}
		if width-fixed >= 36 || (!l.files && !l.delta && !l.when && !l.author) {
			l.subj = imax(8, width-fixed)
			return l
		}
		switch {
		case l.author:
			l.author = false
		case l.when:
			l.when = false
		case l.delta:
			l.delta = false
		default:
			l.files = false
		}
	}
}

func (l colLayout) cells(hash, subj, files, delta, when, author string) string {
	s := fit(hash, 7) + "  " + fit(subj, l.subj)
	if l.files {
		s += "  " + padL(files, 8)
	}
	if l.delta {
		s += "  " + padR(delta, 13)
	}
	if l.when {
		s += "  " + fit(when, 13)
	}
	if l.author {
		s += "  " + fit(author, 14)
	}
	return s
}

type listState struct {
	all    []Commit
	hay    [][]string
	idx    []int
	cur    int
	top    int
	query  string
	typing bool
}

func newList(commits []Commit) *listState {
	st := &listState{all: commits}
	st.hay = make([][]string, len(commits))
	for i, c := range commits {
		parts := []string{strings.ToLower(c.Hash + " " + c.Author + " " + c.Subject)}
		for _, f := range c.Files {
			parts = append(parts, strings.ToLower(f.Path))
		}
		st.hay[i] = parts
	}
	st.refilter()
	return st
}

func (s *listState) refilter() {
	tokens := strings.Fields(strings.ToLower(s.query))
	s.idx = s.idx[:0]
	for i := range s.all {
		ok := true
		for _, t := range tokens {
			found := false
			for _, h := range s.hay[i] {
				if strings.Contains(h, t) {
					found = true
					break
				}
			}
			if !found {
				ok = false
				break
			}
		}
		if ok {
			s.idx = append(s.idx, i)
		}
	}
	s.cur, s.top = 0, 0
}

func (s *listState) clamp(rows int) {
	if s.cur >= len(s.idx) {
		s.cur = len(s.idx) - 1
	}
	if s.cur < 0 {
		s.cur = 0
	}
	if s.cur < s.top {
		s.top = s.cur
	}
	if s.cur >= s.top+rows {
		s.top = s.cur - rows + 1
	}
	if s.top < 0 {
		s.top = 0
	}
}

func (s *listState) render(r *Repo, w, h, rows int) []string {
	now := time.Now()
	right := num(len(s.all)) + " commit"
	if s.query != "" || s.typing {
		q := clean(s.query)
		if s.typing {
			q += ""
		}
		right = fmt.Sprintf(`cari "%s"  %s  %s dari %s`, q, g.sep, num(len(s.idx)), num(len(s.all)))
		if asciiSym {
			right = fmt.Sprintf("cari \"%s\"  -  %s dari %s", q, num(len(s.idx)), num(len(s.all)))
		}
	}
	lay := layoutCols(inner(w))
	lines := []string{
		titleLine(r, "", right, w),
		ruleLine(w),
		"   " + dim + lay.cells("HASH", "PESAN", "FILE", "PERUBAHAN", "WAKTU", "PENULIS") + reset,
	}
	for i := 0; i < rows; i++ {
		pos := s.top + i
		if pos >= len(s.idx) {
			if len(s.idx) == 0 && i == 0 {
				lines = append(lines, bodyLine(dim+"Tidak ada commit yang cocok dengan pencarian."+reset, w))
				continue
			}
			lines = append(lines, "")
			continue
		}
		c := s.all[s.idx[pos]]
		sel := pos == s.cur
		subj := clean(c.Subject)
		if sel {
			subj = bold + subj + reset
		}
		files := dim + fmt.Sprintf("%d file", len(c.Files)) + reset
		delta := ""
		if len(c.Files) == 0 && c.Merge {
			files = dim + "merge" + reset
		} else if len(c.Files) > 0 {
			delta = green + "+" + num(c.Add) + reset + " " + red + g.minus + num(c.Del) + reset
		}
		body := lay.cells(yellow+c.Short+reset, subj, files, delta, dim+relTime(c.When, now)+reset, dim+clean(c.Author)+reset)
		lines = append(lines, rowLine(body, sel, w))
	}
	lines = append(lines, ruleLine(w))
	info := ""
	if len(s.idx) > 0 {
		c := s.all[s.idx[s.cur]]
		info = dim + c.Hash + "  " + g.sep + "  " + clean(c.Author) + "  " + g.sep + "  " + fmtDate(c.When) + reset
	}
	lines = append(lines, bodyLine(info, w))
	pos := ""
	if len(s.idx) > 0 {
		pos = fmt.Sprintf("%s / %s", num(s.cur+1), num(len(s.idx)))
	}
	if s.typing {
		lines = append(lines, keysLine([][2]string{{"Enter", "terapkan"}, {"Esc", "hapus"}}, pos, w))
	} else {
		lines = append(lines, keysLine([][2]string{{g.up + g.down, "pilih"}, {"Enter", "buka"}, {"/", "cari"}, {"PgUp/PgDn", "halaman"}, {"q", "keluar"}}, pos, w))
	}
	return lines
}

func (s *listState) typingKey(k key) {
	switch k.k {
	case kEnter:
		s.typing = false
	case kEsc, kCtrlC:
		s.typing = false
		s.query = ""
		s.refilter()
	case kBack:
		if rs := []rune(s.query); len(rs) > 0 {
			s.query = string(rs[:len(rs)-1])
			s.refilter()
		}
	case kSpace:
		s.query += " "
		s.refilter()
	case kRune:
		s.query += string(k.r)
		s.refilter()
	case kUp:
		s.cur--
	case kDown:
		s.cur++
	}
}

func listScreen(r *Repo, commits []Commit) {
	st := newList(commits)
	for {
		w, h := termSize()
		rows := imax(3, h-6)
		st.clamp(rows)
		draw(st.render(r, w, h, rows), w, h)
		k := nextKey()
		if st.typing {
			st.typingKey(k)
			continue
		}
		switch k.k {
		case kCtrlC:
			return
		case kUp:
			st.cur--
		case kDown:
			st.cur++
		case kPgUp:
			st.cur -= rows
		case kPgDn:
			st.cur += rows
		case kHome:
			st.cur = 0
		case kEnd:
			st.cur = len(st.idx) - 1
		case kEnter, kRight:
			if len(st.idx) > 0 {
				if detailScreen(r, st.all[st.idx[st.cur]]) {
					return
				}
			}
		case kEsc:
			if st.query != "" {
				st.query = ""
				st.refilter()
			}
		case kRune:
			switch k.r {
			case 'q', 'Q':
				return
			case '/':
				st.typing = true
			case 'j':
				st.cur++
			case 'k':
				st.cur--
			case 'g':
				st.cur = 0
			case 'G':
				st.cur = len(st.idx) - 1
			}
		}
	}
}

type detailState struct {
	c      Commit
	cur    int
	top    int
	sel    []bool
	mode   copyMode
	msg    string
	msgErr bool
}

func (d *detailState) clamp(rows int) {
	if d.cur >= len(d.c.Files) {
		d.cur = len(d.c.Files) - 1
	}
	if d.cur < 0 {
		d.cur = 0
	}
	if d.cur < d.top {
		d.top = d.cur
	}
	if d.cur >= d.top+rows {
		d.top = d.cur - rows + 1
	}
	if d.top < 0 {
		d.top = 0
	}
}

func (d *detailState) selected() []File {
	var out []File
	for i, f := range d.c.Files {
		if i < len(d.sel) && d.sel[i] {
			out = append(out, f)
		}
	}
	return out
}

func statusColor(s string) string {
	switch s {
	case "A":
		return green
	case "D":
		return red
	case "M":
		return yellow
	}
	return accent
}

func (d *detailState) render(r *Repo, w, h, rows int) []string {
	c := d.c
	in := inner(w)
	cbW := visLen(g.on)
	pathW := imax(10, in-(cbW+4)-2-15)
	count := 0
	for i, s := range d.sel {
		if i < len(c.Files) && s {
			count++
		}
	}
	meta := dim + clean(c.Author) + "  " + g.sep + "  " + fmtDate(c.When) + "  " + g.sep + "  " + relTime(c.When, time.Now()) + reset
	stats := green + "+" + num(c.Add) + reset + " " + red + g.minus + num(c.Del) + reset + dim + "  " + g.sep + "  " + fmt.Sprintf("%d file", len(c.Files)) + reset
	if c.Merge {
		stats = yellow + "merge" + reset + dim + "  " + g.sep + "  " + fmt.Sprintf("%d file", len(c.Files)) + reset
	}
	lines := []string{
		titleLine(r, yellow+c.Short+reset, "", w),
		ruleLine(w),
		bodyLine(bold+clean(c.Subject)+reset, w),
		"   " + joinLR(meta, stats, in),
		ruleLine(w),
		"   " + dim + strings.Repeat(" ", cbW+4) + padR("BERKAS", pathW) + "  " + padL("TAMBAH", 7) + padL("HAPUS", 8) + reset,
	}
	for i := 0; i < rows; i++ {
		pos := d.top + i
		if pos >= len(c.Files) {
			if len(c.Files) == 0 && i == 0 {
				lines = append(lines, bodyLine(dim+"Commit ini tidak memiliki perubahan berkas."+reset, w))
				continue
			}
			lines = append(lines, "")
			continue
		}
		f := c.Files[pos]
		cb := dim + g.off + reset
		if pos < len(d.sel) && d.sel[pos] {
			cb = accent + g.on + reset
		}
		p := truncLeft(clean(f.Path), pathW)
		dir, base := "", p
		if j := strings.LastIndex(p, "/"); j >= 0 {
			dir, base = p[:j+1], p[j+1:]
		}
		if pos == d.cur {
			base = bold + base + reset
		}
		pathCol := dim + dir + reset + base
		if pos >= len(d.sel) || !d.sel[pos] {
			pathCol = dim + dir + base + reset
		}
		right := padL(green+"+"+num(f.Add)+reset, 7) + padL(red+g.minus+num(f.Del)+reset, 8)
		if f.Binary {
			right = padL(dim+"biner"+reset, 15)
		}
		body := cb + " " + statusColor(f.Status) + f.Status + reset + "  " + padR(pathCol, pathW) + "  " + right
		lines = append(lines, rowLine(body, pos == d.cur, w))
	}
	lines = append(lines, ruleLine(w))
	var modes []string
	for i, n := range modeNames {
		if copyMode(i) == d.mode {
			if useColor {
				modes = append(modes, bold+accent+"\x1b[4m"+n+reset)
			} else {
				modes = append(modes, "["+n+"]")
			}
		} else {
			modes = append(modes, dim+n+reset)
		}
	}
	left := dim + "Salin sebagai" + reset + "   " + strings.Join(modes, "   ")
	rightMsg := dim + fmt.Sprintf("%d dari %d dipilih", count, len(c.Files)) + reset
	if d.msg != "" {
		if d.msgErr {
			rightMsg = red + g.bad + " " + d.msg + reset
		} else {
			rightMsg = green + g.ok + " " + d.msg + reset
		}
	}
	lines = append(lines, "   "+joinLR(left, rightMsg, in))
	lines = append(lines, keysLine([][2]string{{"Spasi", "pilih"}, {"a", "semua"}, {"Tab", "mode"}, {"Enter", "salin"}, {"v", "lihat diff"}, {"Esc", "kembali"}}, "", w))
	return lines
}

func (d *detailState) doCopy(r *Repo) {
	files := d.selected()
	if len(files) == 0 {
		d.msg, d.msgErr = "Belum ada berkas yang dipilih", true
		return
	}
	res := buildCopy(r, d.c, files, d.mode)
	if res.Files == 0 {
		d.msg, d.msgErr = "Tidak ada konten yang bisa disalin untuk mode ini", true
		return
	}
	if err := writeClipboard(res.Text); err != nil {
		d.msg, d.msgErr = err.Error(), true
		return
	}
	d.msg = fmt.Sprintf("%d berkas disalin  %s  %s", res.Files, g.sep, humanSize(len(res.Text)))
	if n := len(res.Skipped); n > 0 {
		d.msg += fmt.Sprintf("  %s  %d dilewati", g.sep, n)
	}
	d.msgErr = false
}

func detailScreen(r *Repo, c Commit) bool {
	d := &detailState{c: c, sel: make([]bool, len(c.Files))}
	for i := range d.sel {
		d.sel[i] = true
	}
	for {
		w, h := termSize()
		rows := imax(3, h-9)
		d.clamp(rows)
		draw(d.render(r, w, h, rows), w, h)
		k := nextKey()
		d.msg = ""
		n := len(c.Files)
		switch k.k {
		case kCtrlC:
			return true
		case kEsc, kBack, kLeft:
			return false
		case kUp:
			d.cur--
		case kDown:
			d.cur++
		case kPgUp:
			d.cur -= rows
		case kPgDn:
			d.cur += rows
		case kHome:
			d.cur = 0
		case kEnd:
			d.cur = n - 1
		case kSpace:
			if n > 0 && d.cur >= 0 && d.cur < n && d.cur < len(d.sel) {
				d.sel[d.cur] = !d.sel[d.cur]
			}
		case kTab:
			d.mode = (d.mode + 1) % copyMode(len(modeNames))
		case kBackTab:
			d.mode = (d.mode + copyMode(len(modeNames)) - 1) % copyMode(len(modeNames))
		case kEnter:
			d.doCopy(r)
		case kRight:
			if n > 0 && d.cur >= 0 && d.cur < n {
				if previewScreen(r, c, c.Files[d.cur]) {
					return true
				}
			}
		case kRune:
			switch k.r {
			case 'q', 'Q':
				return false
			case 'c':
				d.doCopy(r)
			case 'v':
				if n > 0 && d.cur >= 0 && d.cur < n {
					if previewScreen(r, c, c.Files[d.cur]) {
						return true
					}
				}
			case 'a':
				allSelected := true
				for _, s := range d.sel {
					if !s {
						allSelected = false
						break
					}
				}
				for i := range d.sel {
					d.sel[i] = !allSelected
				}
			case 'j':
				d.cur++
			case 'k':
				d.cur--
			case '1', '2', '3', '4':
				d.mode = copyMode(k.r - '1')
			}
		}
	}
}

func colorDiffLines(d string) []string {
	raw := strings.Split(strings.TrimRight(strings.ReplaceAll(d, "\r", ""), "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		l = clean(l)
		switch {
		case strings.HasPrefix(l, "diff --git"), strings.HasPrefix(l, "index "),
			strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"), strings.HasPrefix(l, "new file"),
			strings.HasPrefix(l, "deleted file"):
			out = append(out, dim+l+reset)
		case strings.HasPrefix(l, "@@"):
			out = append(out, accent+l+reset)
		case strings.HasPrefix(l, "+"):
			out = append(out, green+l+reset)
		case strings.HasPrefix(l, "-"):
			out = append(out, red+l+reset)
		default:
			out = append(out, l)
		}
	}
	return out
}

// Tampilan split view seperti GitHub Desktop
func previewScreen(r *Repo, c Commit, f File) bool {
	var oldCode, newCode []string
	
	// Ambil kode lama dan baru
	if f.Status != "A" && c.Parent != "" {
		if old, err := r.show(c.Parent, f.Path); err == nil {
			oldCode = strings.Split(strings.ToValidUTF8(string(old), "\uFFFD"), "\n")
		}
	}
	if f.Status != "D" {
		if nw, err := r.show(c.Hash, f.Path); err == nil {
			newCode = strings.Split(strings.ToValidUTF8(string(nw), "\uFFFD"), "\n")
		}
	}
	
	// Highlight diff
	oldLines := highlightOldCode(oldCode)
	newLines := highlightNewCode(newCode)
	
	viewMode := 0 // 0 = split, 1 = unified diff
	topOld, topNew := 0, 0
	
	for {
		w, h := termSize()
		rows := imax(3, h-8)
		midCol := w / 2
		
		maxTopOld := imax(0, len(oldLines)-rows)
		maxTopNew := imax(0, len(newLines)-rows)
		topOld = imax(0, imin(topOld, maxTopOld))
		topNew = imax(0, imin(topNew, maxTopNew))
		
		lines := []string{
			titleLine(r, yellow+c.Short+reset+dim+" "+g.arrow+" "+reset+clean(f.Path), "", w),
			ruleLine(w),
		}
		
		// Header untuk 2 panel
		header := "   " + bold + "KODE SEBELUMNYA" + reset
		if viewMode == 0 {
			header += strings.Repeat(" ", midCol-visLen(header)-22)
			header += bold + "KODE PERUBAHAN" + reset
		}
		lines = append(lines, "   "+dim+header+reset)
		lines = append(lines, ruleLine(w))
		
		// Render 2 panel side by side
		for i := 0; i < rows; i++ {
			pos := topOld + i
			leftLine := ""
			if pos < len(oldLines) {
				leftLine = oldLines[pos]
			}
			
			if viewMode == 0 {
				// Split view
				posNew := topNew + i
				rightLine := ""
				if posNew < len(newLines) {
					rightLine = newLines[posNew]
				}
				
				leftTrunc := truncate(leftLine, midCol-6)
				rightTrunc := truncate(rightLine, midCol-6)
				
				line := "   " + leftTrunc
				padding := midCol - 6 - visLen(leftTrunc)
				if padding > 0 {
					line += strings.Repeat(" ", padding)
				}
				line += "  " + rightTrunc
				lines = append(lines, line)
			} else {
				// Unified diff (single column)
				lines = append(lines, "   "+oldLines[topOld+i])
			}
		}
		
		lines = append(lines, ruleLine(w))
		
		// Button salin
		copyBtn := "[" + bold + cyan + "C" + reset + dim + "opy" + reset + "]"
		switchBtn := "[" + bold + yellow + "Tab" + reset + dim + "View" + reset + "]"
		
		leftInfo := copyBtn + "   " + switchBtn
		rightInfo := fmt.Sprintf("%d / %d baris", imin(topOld+rows, len(oldLines)), len(oldLines))
		
		lines = append(lines, "   "+joinLR(leftInfo, dim+rightInfo+reset, inner(w)))
		lines = append(lines, keysLine([][2]string{{g.up + g.down, "gulir"}, {"PgUp/PgDn", "halaman"}, {"C", "salin"}, {"Tab", "ubah tampilan"}, {"Esc", "kembali"}}, "", w))
		
		draw(lines, w, h)
		
		switch k := nextKey(); k.k {
		case kCtrlC:
			return true
		case kEsc, kBack, kLeft:
			return false
		case kUp:
			if viewMode == 0 {
				topOld--
				topNew--
			} else {
				topOld--
			}
		case kDown:
			if viewMode == 0 {
				topOld++
				topNew++
			} else {
				topOld++
			}
		case kPgUp:
			topOld -= rows
			topNew -= rows
		case kPgDn, kSpace:
			topOld += rows
			topNew += rows
		case kHome:
			topOld = 0
			topNew = 0
		case kEnd:
			topOld = maxTopOld
			topNew = maxTopNew
		case kTab:
			viewMode = 1 - viewMode
			if viewMode == 1 {
				// Switch to unified diff
				if d, err := r.diff(c, f.Path); err == nil {
					oldLines = colorDiffLines(d)
				}
			} else {
				oldLines = highlightOldCode(oldCode)
			}
		case kRune:
			switch k.r {
			case 'q', 'Q':
				return false
			case 'c', 'C':
				// Salin kode yang ditampilkan
				var textToCopy string
				if viewMode == 0 {
					// Salin kode baru
					textToCopy = strings.Join(newCode, "\n")
				} else {
					// Salin diff
					if d, err := r.diff(c, f.Path); err == nil {
						textToCopy = d
					}
				}
				if err := writeClipboard(textToCopy); err == nil {
					// Show success message briefly
				}
			case 'j':
				topOld++
				topNew++
			case 'k':
				topOld--
				topNew--
			}
		}
	}
}

func highlightOldCode(lines []string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = dim + line + reset
	}
	return result
}

func highlightNewCode(lines []string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = green + line + reset
	}
	return result
}	