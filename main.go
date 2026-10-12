package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

const usage = `codediff — telusuri riwayat commit GitHub dan salin kode lama/baru ke clipboard

Pemakaian:
  codediff                         mode interaktif (akan meminta URL)
  codediff <repo> [opsi]

  <repo>   https://github.com/pemilik/repo, pemilik/repo, atau path repo lokal

Opsi:
  -b, --branch NAMA   branch/tag/ref yang dibaca (default: branch utama)
  -n, --limit N       batasi jumlah commit terbaru yang dimuat
      --no-anim       matikan animasi
      --ascii         pakai simbol ASCII (untuk terminal lama)
      --no-color      matikan warna
  -h, --help          tampilkan bantuan`

type options struct {
	src, branch string
	limit       int
	help        bool
}

func parseArgs(args []string) (options, error) {
	var o options
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("opsi %s membutuhkan nilai", a)
			}
			i++
			return args[i], nil
		}
		switch a {
		case "-h", "--help", "help":
			o.help = true
		case "--no-anim":
			anim = false
		case "--ascii":
			asciiSym = true
		case "--no-color":
			os.Setenv("NO_COLOR", "1")
		case "-b", "--branch":
			v, err := next()
			if err != nil {
				return o, err
			}
			o.branch = v
		case "-n", "--limit":
			v, err := next()
			if err != nil {
				return o, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return o, errors.New("nilai --limit harus berupa angka positif")
			}
			o.limit = n
		default:
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				return o, fmt.Errorf("opsi tidak dikenal: %s", a)
			}
			if o.src != "" {
				return o, errors.New("hanya satu repositori yang dapat diberikan")
			}
			o.src = a
		}
	}
	return o, nil
}

func promptRepo(rd *bufio.Reader) (string, bool) {
	fmt.Println("  " + dim + "Contoh  https://github.com/pemilik/repo   " + g.sep + "   pemilik/repo   " + g.sep + "   path lokal" + reset)
	fmt.Println()
	fmt.Print("  " + accent + "Repositori" + reset + "  " + dim + g.arrow + reset + " ")
	s, err := rd.ReadString('\n')
	s = strings.TrimSpace(s)
	if err != nil && s == "" {
		return "", false
	}
	fmt.Println()
	return s, s != ""
}

func trapInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if leave != nil {
			leave()
		}
		fmt.Print("\x1b[?25h\r\n")
		os.Exit(130)
	}()
}

func main() {
	enableANSI()
	opts, err := parseArgs(os.Args[1:])
	initTheme()
	if err != nil {
		printFail(err.Error())
		fmt.Println()
		fmt.Println(usage)
		os.Exit(2)
	}
	if opts.help {
		fmt.Println(usage)
		return
	}
	trapInterrupt()
	defer func() {
		if leave != nil {
			leave()
		}
	}()

	printSplashScreen()
	rd := bufio.NewReader(os.Stdin)
	var src, name string
	var local bool
	for {
		if opts.src == "" {
			s, ok := promptRepo(rd)
			if !ok {
				return
			}
			opts.src = s
		}
		src, name, local, err = parseSource(opts.src)
		if err == nil {
			break
		}
		printFail(err.Error())
		fmt.Println()
		if len(os.Args) > 1 && os.Args[1] == opts.src {
			os.Exit(1)
		}
		opts.src = ""
	}

	if err := step("Memeriksa git", func(p *prog) (string, error) { return gitVersion() }); err != nil {
		printFail(err.Error())
		os.Exit(1)
	}

	var repo *Repo
	err = step("Menyiapkan repositori", func(p *prog) (string, error) {
		r, err := prepareRepo(src, name, local, opts.branch, p)
		if err != nil {
			return "", err
		}
		if err := resolveBranch(r); err != nil {
			return "", err
		}
		repo = r
		switch {
		case r.Local:
			return "repositori lokal  " + g.sep + "  " + r.Branch, nil
		case r.Stale:
			return "cache lokal, gagal memperbarui  " + g.sep + "  " + r.Branch, nil
		}
		return r.Name + "  " + g.sep + "  " + r.Branch, nil
	})
	if err != nil {
		printFail(err.Error())
		os.Exit(1)
	}

	var commits []Commit
	err = step("Membaca riwayat commit", func(p *prog) (string, error) {
		cs, err := readLog(repo, opts.limit, p)
		if err != nil {
			return "", err
		}
		commits = cs
		files := 0
		for _, c := range cs {
			files += len(c.Files)
		}
		return num(len(cs)) + " commit  " + g.sep + "  " + num(files) + " perubahan berkas", nil
	})
	if err != nil {
		printFail(err.Error())
		os.Exit(1)
	}

	if err := enterTUI(); err != nil {
		printFail(err.Error() + " — jalankan dari terminal interaktif")
		os.Exit(1)
	}
	listScreen(repo, commits)
}