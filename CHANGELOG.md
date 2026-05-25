# Changelog

All notable changes to **go-tagger** will be documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Initial public release

---

## [1.0.0] - 2024-01-01

### Added
- Full ID3v2.3/2.4 tag reader and writer (pure Go, zero external deps)
- True-color hex terminal output with perceptual gradient support
- Emoji-rich terminal UI (fully disableable via `-nc` / `NO_COLOR`)
- Batch processing of entire directories, with recursive mode (`-rec`)
- Title file import — parse `NN. Title` lines from a plain text file
- Cover art set from JPEG/PNG/GIF/BMP (`-C`) and extract (`-ec`)
- Smart file rename: by tag (`-R title`) or by filename (`-R file`)
- Dry-run/test mode (`-T`) — shows all changes without writing
- Atomic file saves via temp-file rename — no half-written files
- Cross-platform: Linux, macOS, Windows, FreeBSD; amd64/arm64/386/arm
- GitHub Actions CI (test on Linux/macOS/Windows × Go 1.21+1.22)
- GitHub Actions Release (9 platforms, sha256 checksums, auto changelog)
