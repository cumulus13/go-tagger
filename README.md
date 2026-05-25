# 🎵 go-tagger

> **MP3 ID3v2 Tag Editor** — batch-edit, rename, cover art, true-color hex output, and emoji-rich terminal UI. Zero external dependencies.

[![CI](https://github.com/cumulus13/go-tagger/actions/workflows/ci.yml/badge.svg)](https://github.com/cumulus13/go-tagger/actions/workflows/ci.yml)
[![Release](https://github.com/cumulus13/go-tagger/actions/workflows/release.yml/badge.svg)](https://github.com/cumulus13/go-tagger/releases)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-blue.svg)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## ✨ Features

| Feature | Detail |
|---|---|
| 🏷️ **Full ID3v2 tag support** | Title, Artist, Album, Track, Disc, Genre, Date, BPM, Key, ISRC, Barcode, Composer, Lyric, URL, Cover art, and more |
| 🎨 **True-color hex output** | Every change displayed with `#rrggbb` RGB colors and perceptual color gradients |
| 😄 **Emoji UI** | Rich emoji feedback for every operation (`💾 Saved`, `⚠️ Warning`, `✅ OK`) |
| 📁 **Batch processing** | Tag whole directories with a single command; recursive mode available |
| 📝 **Title file import** | Provide a `.txt` list (`01. Title`, …) — track numbers and titles auto-parsed |
| 🖼️ **Cover art** | Set from JPEG/PNG/GIF/BMP file or extract embedded art |
| ✏️ **Smart rename** | Rename files by tag or by filename pattern |
| 🧪 **Dry-run mode** | Preview every change before writing with `-T` |
| 🔒 **Atomic saves** | All writes go through a temp-file rename — no half-written files |
| ⚡ **Zero deps** | Pure Go standard library only; no CGO, no external packages |
| 🧰 **Cross-platform** | Linux, macOS, Windows, FreeBSD; amd64/arm64/386/arm |

---

## 📦 Installation

### Pre-built binary (recommended)

Download the latest release for your platform from the [Releases page](https://github.com/cumulus13/go-tagger/releases):

```bash
# Linux amd64
curl -L https://github.com/cumulus13/go-tagger/releases/latest/download/go-tagger_linux_amd64.tar.gz | tar xz
sudo mv go-tagger /usr/local/bin/

# macOS arm64 (Apple Silicon)
curl -L https://github.com/cumulus13/go-tagger/releases/latest/download/go-tagger_darwin_arm64.tar.gz | tar xz
sudo mv go-tagger /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/cumulus13/go-tagger.git
cd go-tagger
make build            # current platform
make install          # install to $GOPATH/bin
make dist             # all platforms → dist/
```

> **Requirements:** Go 1.21+. No other dependencies.

---

## 🚀 Quick Start

```bash
# Tag a single file
go-tagger song.mp3 -a "Artist Name" -aa "Album Title" -g "Pop" -dd 2024

# Tag all MP3s in a directory using a title list
go-tagger ./album/ -t titles.txt -a "Artist" -aa "Album" -d 1

# Show all tags on a file
go-tagger song.mp3 -I

# Dry-run: preview what would change
go-tagger ./album/ -a "New Artist" -T

# Extract embedded cover art
go-tagger song.mp3 -ec

# Rename all files in a directory to "NN. Title.mp3"
go-tagger ./album/ -R title
```

---

## 📋 All Flags

### Tag Fields

| Flag | Description |
|------|-------------|
| `-t <title\|file.txt\|'file'>` | Set title(s). Inline value, path to `.txt` file (one title per line), or `file` to read from existing tag |
| `-tt <track>` | Set track number. Accepts `3`, `03`, or `03/12` |
| `-ltt <n>` | Override the total track count used in `NN/TT` formatting |
| `-ct` | Update the total in an existing `NN/TT` track tag to the file count |
| `-a <artist>` | Set Artist (`TPE1`) |
| `-sa <artist>` | Set Solo Artist (`TXXX:ARTIST`) |
| `-aa <album>` | Set Album (`TALB`) |
| `-aaa <artist>` | Set Album Artist (`TPE2`) |
| `-d <disc>` | Set Disc number. Accepts `1`, `01`, or `01/02` |
| `-c <composer>` | Set Composer (`TCOM`) |
| `-o <artist>` | Set Original Artist (`TOPE`) |
| `-cc <comment>` | Set Comment (`COMM`) |
| `-cd <desc>` | Set Comment description field |
| `-i <isrc>` | Set ISRC number (`TSRC`) |
| `-b <barcode>` | Set Barcode (`TXXX:BARCODE`) |
| `-g <genre>` | Set Genre (`TCON`) |
| `-dd <year>` | Set Date/Year (`TDRC`) |
| `-bb <bpm>` | Set BPM (`TBPM`) |
| `-ccc <text>` | Set Copyright (`TCOP`) |
| `-e <text>` | Set Encoded By (`TENC`) |
| `-k <key>` | Set Key (`TKEY`) |
| `-l <text\|file.txt>` | Set Lyric — inline text or path to `.txt` file (`USLT`) |
| `-ln <name>` | Set Lyric Name (`TEXT`) |
| `-r <text>` | Set Remix / Interpreted By (`TPE4`) |
| `-s <text>` | Set Subtitle (`TIT3`) |
| `-u <url>` | Set URL (`WXXX`) |
| `-gg <group>` | Set Content Group (`TIT1`) |
| `-p <publisher>` | Set Publisher (`TPUB`) |
| `-ll <ms>` | Set Length in milliseconds (`TLEN`) |
| `-C <image>` | Set Cover art from image path (JPEG, PNG, GIF, BMP) |
| `-Cn <name>` | Set Cover name label (`TXXX:_cover`). Default: `Cover Album Front` |

### Action Flags

| Flag | Description |
|------|-------------|
| `-gt` | Print track number and title for each file |
| `-I` | Print full ID3 tag info for each file |
| `-ec` | Extract embedded cover art to `Cover.<ext>` |
| `-T` | **Dry-run** — show all changes without writing |
| `-R <title\|file>` | Rename files: `title` uses tag data, `file` parses filename |
| `-RP <pattern>` | Rename separator pattern: `-` for `NN - Title` style |
| `-rec` | Scan directories recursively |
| `-nc` | Disable color output (also honors `NO_COLOR` env var) |
| `-od <dir>` | Output directory for extracted cover art |
| `-v` | Print version and exit |

---

## 📝 Title File Format

When supplying `-t titles.txt`, each line can be:

```
01. First Song Title
02. Second Song Title
03. Third Song Title
```

Track numbers and titles are auto-extracted. Titles without a number prefix are used as-is.
Lines are matched **in file-system sort order** against the MP3 files.

---

## 🎨 Color & Emoji Output

go-tagger uses **24-bit true-color** ANSI escape codes with a [Tokyo Night](https://github.com/folke/tokyonight.nvim) inspired palette:

```
📄 [1/12] /music/album/01. First Song.mp3
──────────────────────────────────────────────────────────────────────────
TITLE:              Old Title → New Title
ARTIST:             Old Artist → New Artist
ALBUM:              Old Album → New Album
💾 Saved
──────────────────────────────────────────────────────────────────────────
```

To disable all color: `go-tagger -nc` or set `NO_COLOR=1` (respects the [no-color.org](https://no-color.org) standard).

To disable emoji: `go-tagger -nc` (emoji and color share the same flag — terminals that don't support emoji typically also don't support true color).

---

## 🔧 Clearing a Tag Field

Set any field to the literal value `clear` to delete it:

```bash
go-tagger song.mp3 -cc clear      # clear comment
go-tagger song.mp3 -t clear       # clear title
go-tagger song.mp3 -i clear       # clear ISRC
```

---

## 🧪 Development

```bash
# Run all tests
make test

# Verbose tests
make test-verbose

# Coverage report → coverage.html
make test-cover

# Format + vet + test + build
make all

# Cross-compile for all platforms
make dist

# Show all make targets
make help
```

---

## 🔄 Releasing

1. Tag a commit: `git tag v1.2.3 && git push origin v1.2.3`
2. The [release workflow](.github/workflows/release.yml) automatically:
   - Cross-compiles for 9 platform/arch combinations
   - Packages `.tar.gz` (Linux/macOS/FreeBSD) and `.zip` (Windows)
   - Generates `SHA256SUMS`
   - Publishes a GitHub Release with all assets

---

## 📖 ID3v2 Frame Reference

| Frame | Field |
|---|---|
| `TIT1` | Content Group |
| `TIT2` | Title |
| `TIT3` | Subtitle |
| `TPE1` | Artist |
| `TPE2` | Album Artist |
| `TPE4` | Remix/Interpreted By |
| `TOPE` | Original Artist |
| `TALB` | Album |
| `TRCK` | Track (`NN/TT`) |
| `TPOS` | Disc (`NN/TT`) |
| `TCON` | Genre |
| `TDRC` | Date/Year |
| `TBPM` | BPM |
| `TKEY` | Key |
| `TSRC` | ISRC |
| `TCOP` | Copyright |
| `TENC` | Encoded By |
| `TLEN` | Length (ms) |
| `TPUB` | Publisher |
| `TCOM` | Composer |
| `TEXT` | Lyric Name |
| `COMM` | Comment |
| `USLT` | Unsynchronised Lyrics |
| `WXXX` | URL |
| `APIC` | Cover Art |
| `TXXX:BARCODE` | Barcode |
| `TXXX:ARTIST` | Solo Artist |
| `TXXX:_cover` | Cover Name |

---

## 📄 License

[MIT](LICENSE) © 2026 Hadi Cahyadi

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)