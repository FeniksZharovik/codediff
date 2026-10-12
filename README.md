<div align="center">

# 🔍 codediff

**Telusuri commit GitHub, bandingkan kode lama vs baru, dan salin ke clipboard, semuanya dari terminal.**

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)
![Dependencies](https://img.shields.io/badge/dependencies-zero-brightgreen)
![License](https://img.shields.io/badge/license-MIT-blue)

[Mulai Cepat](#-mulai-cepat-3-langkah) • [Instalasi](#-instalasi) • [Cara Pakai](#-cara-penggunaan) • [Shortcut](#-panduan-keyboard-shortcuts) • [Troubleshooting](#-troubleshooting)

</div>

---

## 📖 Apa itu codediff?

**codediff** adalah aplikasi terminal (CLI/TUI) yang cepat dan ringan untuk menelusuri riwayat commit GitHub, membandingkan perubahan kode (lama vs baru), dan menyalin potongan kode langsung ke clipboard.

Dibangun dengan **Go**, berjalan di **Windows, macOS, dan Linux** tanpa dependensi eksternal selain **Git**.

---

## ✨ Fitur Utama

| | Fitur | Keterangan |
|---|---|---|
| 🕵️ | **Penelusuran Commit** | Riwayat commit lengkap dengan *fuzzy search* real-time |
| 📊 | **Tampilan ala GitHub** | Mode *Split View* (berdampingan) atau *Unified Diff*, hijau = tambahan, merah = hapusan |
| ☑️ | **Pilih File Fleksibel** | Checkbox per file, atau pilih semua file dalam satu commit |
| 📋 | **Salin ke Clipboard** | Kode lama, kode baru, keduanya, atau diff mentah, cukup satu tombol |
| 💾 | **Cache Lokal** | Repo di-cache, penelusuran berikutnya sangat cepat dan bisa offline |
| 🎨 | **Tampilan ANSI Modern** | Warna, animasi loading, simbol Unicode (fallback ASCII untuk terminal lama) |

---

## 🚀 Mulai Cepat (3 Langkah)

```bash
# 1️⃣ Pastikan Git terinstal
git --version

# 2️⃣ Build codediff
go build -o codediff -ldflags="-s -w" .

# 3️⃣ Jalankan!
./codediff
```

> 💡 **Tips:** Di Windows, gunakan `codediff.exe`.

---

## 📦 Instalasi

### ✅ Prasyarat

- **Git** terinstal dan bisa diakses dari terminal (`git --version` harus berhasil)
- **Go 1.21+** (hanya jika ingin build dari source)

### Pilih metode instalasi Anda

<details>
<summary><b>🪟 Opsi 1: Pakai file executable (Windows)</b></summary>

<br>

Jika sudah punya `codediff.exe`, pindahkan ke folder yang ada di **PATH** sistem, atau jalankan langsung dari folder saat ini:

```powershell
.\codediff.exe
```

</details>

<details>
<summary><b>🛠️ Opsi 2: Build dari source (Windows / macOS / Linux)</b></summary>

<br>

```bash
# Masuk ke direktori proyek
cd codediff

# Windows -> menghasilkan codediff.exe
go build -o codediff.exe -ldflags="-s -w" .

# macOS / Linux -> menghasilkan codediff
go build -o codediff -ldflags="-s -w" .
```

Jalankan:

```bash
# Windows
.\codediff.exe

# macOS / Linux
./codediff
```

</details>

---

## 🎮 Cara Penggunaan

### 1️⃣ Mode Interaktif (Direkomendasikan)

Jalankan tanpa argumen, lalu ikuti petunjuk di layar untuk memasukkan URL atau nama repositori.

```bash
codediff
```

### 2️⃣ Langsung lewat URL atau Nama Repo

```bash
# Menggunakan URL lengkap
codediff https://github.com/user/repo

# Menggunakan format singkat (owner/repo)
codediff owner/repo

# Menggunakan repositori lokal di komputer Anda
codediff /path/ke/folder/project
```

### 3️⃣ Opsi Tambahan (Flags)

| Flag | Deskripsi |
|------|-----------|
| `--branch <nama>`, `-b` | Tentukan branch, tag, atau commit spesifik (default: branch utama) |
| `--limit <angka>`, `-l` | Batasi jumlah commit yang dimuat |
| `--no-cache` | Abaikan cache lokal dan ambil data terbaru dari remote |
| `--ascii` | Gunakan simbol ASCII (bukan Unicode), cocok untuk terminal lama |
| `--no-color` | Matikan output berwarna |
| `--help`, `-h` | Tampilkan bantuan penggunaan |

**Contoh kombinasi:**

```bash
codediff owner/repo --branch develop --limit 50
codediff owner/repo --no-cache --ascii
```

---

## ⌨️ Panduan Keyboard (Shortcuts)

Aplikasi ini sepenuhnya dikendalikan lewat keyboard. Ada **3 layar** utama:

```
📜 Daftar Commit  ──Enter──▶  📂 Detail Commit  ──Enter──▶  👁️ Preview Kode
        ◀──Esc──                     ◀──Esc──
```

### 📜 Layar Daftar Commit

| Tombol | Fungsi |
|--------|--------|
| `↑` / `↓` atau `j` / `k` | Navigasi daftar commit |
| `Enter` atau `→` | Buka detail commit yang dipilih |
| `/` | Aktifkan mode pencarian (*fuzzy search*) |
| `Esc` | Batalkan pencarian |
| `g` / `G` | Lompat ke commit paling awal / paling baru |
| `q` | Keluar dari aplikasi |

### 📂 Layar Detail Commit (File List & Checkbox)

| Tombol | Fungsi |
|--------|--------|
| `↑` / `↓` | Navigasi daftar file |
| `Space` | Centang / hapus centang file yang sedang dipilih |
| `a` | Pilih semua file / batalkan semua pilihan |
| `v` atau `Enter` | Lihat preview perubahan file (Split View / Diff) |
| `Tab` | Ganti mode penyalinan: **Kode Lama → Kode Baru → Keduanya → Diff** |
| `c` | Salin file yang dicentang ke clipboard sesuai mode yang dipilih |
| `Esc` | Kembali ke daftar commit |

### 👁️ Layar Preview Kode (Code Preview)

| Tombol | Fungsi |
|--------|--------|
| `↑` / `↓` | Scroll kode ke atas / bawah |
| `PgUp` / `PgDn` | Scroll satu halaman penuh |
| `Tab` | Ganti tampilan: **Split View** (Kiri: Lama, Kanan: Baru) ↔ **Unified Diff** |
| `c` | Salin kode yang sedang ditampilkan di layar ke clipboard |
| `Esc` | Kembali ke daftar file |

---

## 📋 Cara Kerja Fitur "Salin" (Copy)

Di layar Detail Commit, tekan `Tab` untuk memilih **mode penyalinan**. Mode aktif akan ditandai dengan garis bawah. Saat Anda menekan `c`, aplikasi akan menyalin:

| Mode | Yang disalin |
|------|--------------|
| 🔴 **Kode Lama** | Isi file sebelum perubahan (commit sebelumnya) |
| 🟢 **Kode Baru** | Isi file setelah perubahan di commit yang dipilih |
| 🔵 **Keduanya** | Kode lama dan kode baru, digabung dengan pembatas yang jelas |
| 📝 **Diff** | Format diff mentah (seperti output `git diff`), cocok untuk code review atau patch |

> 📌 **Catatan:** File yang tidak dicentang (checkbox kosong) akan diabaikan saat proses penyalinan.

---

## 🩺 Troubleshooting

<details>
<summary><b>❌ "git tidak ditemukan"</b></summary>

<br>

Pastikan Git sudah terinstal.

- **Windows:** jalankan `winget install Git.Git`, atau unduh dari [git-scm.com](https://git-scm.com)
- Setelah instalasi, pastikan Git ada di **Environment Variable PATH**, lalu buka ulang terminal

</details>

<details>
<summary><b>🔒 "Repositori tidak ditemukan atau bersifat privat"</b></summary>

<br>

Untuk repo privat, pastikan Anda sudah mengatur **SSH key** atau **Git Credential Manager** di komputer Anda, sehingga Git bisa mengakses repo tersebut tanpa meminta password.

Cek dengan:

```bash
git ls-remote https://github.com/owner/repo
```

Jika perintah ini berhasil tanpa error, codediff juga akan bisa mengaksesnya.

</details>

<details>
<summary><b>🔤 Tampilan simbol berantakan / karakter aneh di terminal Windows CMD</b></summary>

<br>

Gunakan **Windows Terminal** atau **PowerShell modern**. Jika masih bermasalah, jalankan dengan mode ASCII:

```bash
codediff --ascii
```

</details>

<details>
<summary><b>📋 Gagal menyalin ke clipboard</b></summary>

<br>

- **Windows:** aplikasi memakai perintah `clip` bawaan
- **Linux:** instal salah satu dari `xclip`, `xsel`, atau `wl-clipboard` (untuk Wayland)

```bash
# Debian / Ubuntu
sudo apt install xclip
```

</details>

---

## 🧰 Teknologi yang Digunakan

- **Bahasa:** Go (Golang) 1.21+
- **Dependensi:** nol, hanya library standar Go (*zero external dependencies*)
- **Sistem:** menjalankan perintah `git` secara langsung untuk membaca riwayat dan diff
- **Tampilan:** ANSI escape code murni untuk warna dan layout terminal

---

## 🤝 Kontribusi

Saran dan perbaikan sangat diterima!

1. **Fork** repositori ini
2. Buat branch fitur: `git checkout -b fitur-keren`
3. **Commit** perubahan: `git commit -m "Tambah fitur keren"`
4. **Push** ke branch: `git push origin fitur-keren`
5. Buka **Pull Request**

---

## 📄 Lisensi

Didistribusikan di bawah **MIT License**. Lihat file [`LICENSE`](LICENSE) untuk informasi lebih lanjut.

---

<div align="center">

⭐ Kalau codediff membantu, jangan lupa beri **star** di repositori ini!

</div>