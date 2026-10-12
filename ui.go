package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	reset, bold, dim, accent, green, red, yellow, subtle, selBg string
	cyan, magenta, orange, navy, white                          string
	ansiRe                                                      = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	anim                                                        = true
	asciiSym                                                    = false
	useColor                                                    = true
)

type glyphs struct {
	spin                                               []string
	ok, bad, pend, barOn, barOff, mark, on, off, minus string
	ell, arrow, sep, rule, up, down                    string
}

var g glyphs

func rgb(r, gr, b int) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, gr, b) }

func initTheme() {
	if os.Getenv("NO_COLOR") != "" {
		useColor = false
	}
	if useColor {
		reset = "\x1b[0m"
		bold = "\x1b[1m"
		dim = rgb(125, 133, 146)
		subtle = rgb(62, 68, 80)
		accent = rgb(97, 175, 239)
		green = rgb(152, 195, 121)
		red = rgb(224, 108, 117)
		yellow = rgb(209, 154, 102)
		cyan = rgb(86, 180, 233)
		magenta = rgb(198, 120, 221)
		orange = rgb(229, 192, 123)
		selBg = "\x1b[48;2;36;41;54m"
		// Palette taken from the Neovim-style reference image:
		// deep royal blue frame lines, bright cyan wordmark, soft white body text.
		navy = rgb(45, 70, 165)
		white = rgb(206, 206, 206)
	}
	if asciiSym {
		g = glyphs{
			spin: []string{"|", "/", "-", "\\"},
			ok:   "OK", bad: "!!", pend: "..", barOn: "#", barOff: "-", mark: ">",
			on: "[x]", off: "[ ]", minus: "-", ell: "...", arrow: ">", sep: "-", rule: "-",
			up: "^", down: "v",
		}
		return
	}
	g = glyphs{
		spin: []string{"⠋", "⠙", "⠹", "", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		ok:   "✓", bad: "✕", pend: "·", barOn: "━", barOff: "─", mark: "▌",
		on: "■", off: "□", minus: "−", ell: "…", arrow: "›", sep: "·", rule: "─",
		up: "↑", down: "↓",
	}
}

func runeWidth(r rune) int {
	switch {
	case r == 0, r < 32, r >= 0x7f && r < 0xa0:
		return 0
	case r >= 0x300 && r <= 0x36f, r >= 0x200b && r <= 0x200f, r == 0xfe0f:
		return 0
	case r >= 0x1100 && r <= 0x115f, r >= 0x2e80 && r <= 0xa4cf, r >= 0xac00 && r <= 0xd7a3,
		r >= 0xf900 && r <= 0xfaff, r >= 0xfe30 && r <= 0xfe6f, r >= 0xff00 && r <= 0xff60,
		r >= 0xffe0 && r <= 0xffe6, r >= 0x1f300 && r <= 0x1f64f, r >= 0x1f900 && r <= 0x1f9ff,
		r >= 0x20000 && r <= 0x3fffd:
		return 2
	}
	return 1
}

func visLen(s string) int {
	n := 0
	for _, r := range ansiRe.ReplaceAllString(s, "") {
		n += runeWidth(r)
	}
	return n
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if visLen(s) <= w {
		return s
	}
	limit := w - visLen(g.ell)
	if limit < 0 {
		limit = 0
	}
	var sb strings.Builder
	cur := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if loc := ansiRe.FindStringIndex(s[i:]); loc != nil && loc[0] == 0 {
				sb.WriteString(s[i : i+loc[1]])
				i += loc[1]
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runeWidth(r)
		if cur+rw > limit {
			break
		}
		sb.WriteString(s[i : i+size])
		cur += rw
		i += size
	}
	sb.WriteString(reset)
	sb.WriteString(g.ell)
	return sb.String()
}

func truncLeft(s string, w int) string {
	rs := []rune(s)
	if w <= 0 {
		return ""
	}
	total := 0
	for _, r := range rs {
		total += runeWidth(r)
	}
	if total <= w {
		return s
	}
	keep := w - visLen(g.ell)
	out := []rune{}
	cur := 0
	for i := len(rs) - 1; i >= 0; i-- {
		rw := runeWidth(rs[i])
		if cur+rw > keep {
			break
		}
		out = append([]rune{rs[i]}, out...)
		cur += rw
	}
	return g.ell + string(out)
}

func padR(s string, w int) string {
	if n := w - visLen(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func padL(s string, w int) string {
	if n := w - visLen(s); n > 0 {
		return strings.Repeat(" ", n) + s
	}
	return s
}

func fit(s string, w int) string { return padR(truncate(s, w), w) }

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func num(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	return string(out)
}

func humanSize(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return strings.Replace(fmt.Sprintf("%.1f KB", float64(n)/1024), ".", ",", 1)
	}
	return strings.Replace(fmt.Sprintf("%.1f MB", float64(n)/1048576), ".", ",", 1)
}

func fmtDur(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%d ms", d.Milliseconds())
	case d < time.Minute:
		return strings.Replace(fmt.Sprintf("%.1f dtk", d.Seconds()), ".", ",", 1)
	}
	return fmt.Sprintf("%d mnt %02d dtk", int(d.Minutes()), int(d.Seconds())%60)
}

var months = []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func fmtDate(t time.Time) string {
	t = t.Local()
	return fmt.Sprintf("%02d %s %d %02d:%02d", t.Day(), months[t.Month()-1], t.Year(), t.Hour(), t.Minute())
}

func relTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "baru saja"
	case d < time.Hour:
		return fmt.Sprintf("%d menit lalu", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d jam lalu", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d hari lalu", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%d minggu lalu", int(d.Hours()/24/7))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d bulan lalu", int(d.Hours()/24/30))
	}
	return fmt.Sprintf("%d tahun lalu", int(d.Hours()/24/365))
}

func rule(n int) string {
	if n < 0 {
		n = 0
	}
	return subtle + strings.Repeat(g.rule, n) + reset
}

func bar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	on := int(pct * float64(width) / 100)
	return accent + strings.Repeat(g.barOn, on) + reset + subtle + strings.Repeat(g.barOff, width-on) + reset
}

func withBg(s, bg string) string {
	if reset == "" {
		return s
	}
	return bg + strings.ReplaceAll(s, reset, reset+bg) + reset
}

func keyHint(k, label string) string {
	return accent + k + reset + " " + dim + label + reset
}

type prog struct {
	mu     sync.Mutex
	detail string
	pct    float64
}

func (p *prog) set(detail string, pct float64) {
	p.mu.Lock()
	p.detail = detail
	p.pct = pct
	p.mu.Unlock()
}

func step(label string, fn func(p *prog) (string, error)) error {
	p := &prog{pct: -1}
	start := time.Now()
	if !anim {
		fmt.Printf("  %s %s\n", g.pend, label)
		res, err := fn(p)
		if err != nil {
			fmt.Printf("  %s %s\n", red+g.bad+reset, label)
			return err
		}
		fmt.Printf("  %s %s  %s\n", green+g.ok+reset, label, dim+res+reset)
		return nil
	}
	w, _ := termSize()
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(80 * time.Millisecond)
		defer t.Stop()
		for i := 0; ; i++ {
			p.mu.Lock()
			d, pc := p.detail, p.pct
			p.mu.Unlock()
			line := "  " + accent + g.spin[i%len(g.spin)] + reset + " " + padR(label, 28)
			if pc >= 0 {
				line += " " + bar(pc, 22) + fmt.Sprintf(" %3d%%", int(pc))
			}
			if d != "" {
				line += "  " + dim + d + reset
			}
			fmt.Print("\r\x1b[2K" + truncate(line, w-1))
			select {
			case <-done:
				return
			case <-t.C:
			}
		}
	}()
	res, err := fn(p)
	close(done)
	wg.Wait()
	if err != nil {
		fmt.Printf("\r\x1b[2K  %s %s\n", red+g.bad+reset, label)
		return err
	}
	line := "  " + green + g.ok + reset + " " + padR(label, 28) + " " + dim + res + "  " + g.sep + "  " + fmtDur(time.Since(start)) + reset
	fmt.Print("\r\x1b[2K" + truncate(line, w-1) + "\n")
	return nil
}

func printFail(msg string) {
	fmt.Println("  " + red + g.bad + reset + " " + msg)
}

func printSplashScreen() {
	// CODEDIFF wordmark, ANSI Shadow block style (same family as the old art).
	lines := []string{
		"  ██████╗  ██████╗ ██████╗ ███████╗██████╗ ██╗███████╗███████╗",
		" ██╔════╝ ██╔═══██╗██╔══██╗██╔════╝██╔══██╗██║██╔════╝██╔════╝",
		" ██║      ██║   ██║██║  ██║█████╗  ██║  ██║██║█████╗  █████╗  ",
		" ██║      ██║   ██║██║  ██║██╔══╝  ██║  ██║██║██╔══╝  ██╔══╝  ",
		" ╚██████╗ ╚██████╔╝██████╔╝███████╗██████╔╝██║██║     ██║     ",
		"  ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝╚═════╝ ╚═╝██║     ██║     ",
	}

	// Vertical cyan glow, matching the bright-cyan wordmark in the
	// reference image: light highlight on top, bright core, teal shading
	// at the bottom edge.
	rowColors := []string{
		rgb(150, 250, 250),
		rgb(0, 236, 236),
		rgb(0, 224, 224),
		rgb(0, 224, 224),
		rgb(0, 176, 176),
		rgb(0, 132, 132),
	}

	const inner = 65 // box interior width (61-col art + 2-space padding)
	top := navy + "╔" + strings.Repeat("═", inner) + "╗" + reset
	sep := navy + "╠" + strings.Repeat("═", inner) + "╣" + reset
	bot := navy + "╚" + strings.Repeat("═", inner) + "╝" + reset
	row := func(content string) string {
		pad := inner - visLen(content)
		if pad < 0 {
			pad = 0
		}
		return navy + "║" + reset + content + strings.Repeat(" ", pad) + navy + "║" + reset
	}
	blank := row("")

	fmt.Println()
	fmt.Println("  " + top)
	fmt.Println("  " + blank)
	for i, line := range lines {
		fmt.Println("  " + row("  "+rowColors[i]+line+reset))
	}
	fmt.Println("  " + blank)
	fmt.Println("  " + sep)
	fmt.Println("  " + row("  "+bold+white+"GitHub Commit Browser & Code Comparator"+reset))
	fmt.Println("  " + row("  " + dim + "v1.0  ·  Written in Go  ·  No dependencies" + reset))
	fmt.Println("  " + sep)
	fmt.Println("  " + row("  "+cyan+"[+]"+reset+" "+dim+"Browse full commit history with search"+reset))
	fmt.Println("  " + row("  "+magenta+"[>]"+reset+" "+dim+"Compare old vs new code side-by-side"+reset))
	fmt.Println("  " + row("  "+cyan+"[>]"+reset+" "+dim+"GitHub-style diff with green/red highlights"+reset))
	fmt.Println("  " + row("  "+magenta+"[>]"+reset+" "+dim+"Select files with checkbox & copy to clipboard"+reset))
	fmt.Println("  " + row("  "+cyan+"[>]"+reset+" "+dim+"Works on Windows, macOS, and Linux"+reset))
	fmt.Println("  " + blank)
	fmt.Println("  " + bot)
	fmt.Println()
}

func printWordmark() {
	printSplashScreen()
}