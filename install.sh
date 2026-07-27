#!/bin/sh
# go-tagger install script
# Handles Android noexec filesystem restrictions automatically.
# Usage:  sh install.sh [version]
# Example: sh install.sh v1.0.3
#
# Author: Hadi Cahyadi <cumulus13@gmail.com>
# Repo:   https://github.com/cumulus13/go-tagger

set -e

REPO="cumulus13/go-tagger"
BINARY_NAME="go-tagger"

# ── Detect version ────────────────────────────────────────────────────────────
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Fetching latest release tag..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi
if [ -z "$VERSION" ]; then
    echo "ERROR: could not determine version. Pass it explicitly: sh install.sh v1.0.3"
    exit 1
fi
echo "Installing go-tagger ${VERSION}"

# ── Detect OS and architecture ────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
KERNEL=$(uname -r | tr '[:upper:]' '[:lower:]')

# Normalise OS
case "$OS" in
    linux)
        # Distinguish Android from regular Linux
        if echo "$KERNEL" | grep -qi "android"; then
            PLATFORM="android"
        else
            PLATFORM="linux"
        fi
        ;;
    darwin)  PLATFORM="darwin"  ;;
    freebsd) PLATFORM="freebsd" ;;
    netbsd)  PLATFORM="netbsd"  ;;
    openbsd) PLATFORM="openbsd" ;;
    mingw*|msys*|cygwin*) PLATFORM="windows" ;;
    *)
        echo "ERROR: unsupported OS: $OS"
        exit 1
        ;;
esac

# Normalise arch
case "$ARCH" in
    x86_64|amd64)   ARCH_SUFFIX="amd64"  ;;
    aarch64|arm64)  ARCH_SUFFIX="arm64"  ;;
    armv7*|armv7l)  ARCH_SUFFIX="armv7"  ;;
    armv6*|armv6l)  ARCH_SUFFIX="armv6"  ;;
    i386|i686)      ARCH_SUFFIX="386"    ;;
    mips64le)       ARCH_SUFFIX="mips64le" ;;
    mips64)         ARCH_SUFFIX="mips64"   ;;
    mipsle)         ARCH_SUFFIX="mipsle"   ;;
    mips)           ARCH_SUFFIX="mips"     ;;
    ppc64le)        ARCH_SUFFIX="ppc64le"  ;;
    riscv64)        ARCH_SUFFIX="riscv64"  ;;
    s390x)          ARCH_SUFFIX="s390x"    ;;
    *)
        echo "ERROR: unsupported architecture: $ARCH"
        exit 1
        ;;
esac

SUFFIX="${PLATFORM}_${ARCH_SUFFIX}"
EXT=""
[ "$PLATFORM" = "windows" ] && EXT=".exe"

ARCHIVE_BASE="go-tagger_${VERSION}_${SUFFIX}"
if [ "$PLATFORM" = "windows" ]; then
    ARCHIVE="${ARCHIVE_BASE}.zip"
else
    ARCHIVE="${ARCHIVE_BASE}.tar.gz"
fi
BINARY="${ARCHIVE_BASE}${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Platform : ${PLATFORM}/${ARCH_SUFFIX}  (uname: ${OS}/${ARCH})"
echo "Archive  : ${ARCHIVE}"
echo "URL      : ${URL}"
echo ""

# ── Determine install path ────────────────────────────────────────────────────
# Android: /storage/ and /sdcard/ are noexec — NEVER install there.
# Priority: Termux $PREFIX/bin > /data/local/tmp > /usr/local/bin > ~/bin
find_install_dir() {
    # Termux
    if [ -n "${PREFIX:-}" ] && [ -d "${PREFIX}/bin" ]; then
        echo "${PREFIX}/bin"
        return
    fi
    # Termux (fallback detection)
    if [ -d "/data/data/com.termux/files/usr/bin" ]; then
        echo "/data/data/com.termux/files/usr/bin"
        return
    fi
    # Android fallback (noexec-safe, no root needed)
    if echo "$KERNEL" | grep -qi "android"; then
        echo "/data/local/tmp"
        return
    fi
    # Standard Unix
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi
    if [ -d "$HOME/.local/bin" ]; then
        echo "$HOME/.local/bin"
        return
    fi
    mkdir -p "$HOME/bin"
    echo "$HOME/bin"
}

INSTALL_DIR=$(find_install_dir)
INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}${EXT}"

echo "Install to: ${INSTALL_PATH}"

# Warn if installing to /data/local/tmp
if [ "$INSTALL_DIR" = "/data/local/tmp" ]; then
    echo ""
    echo "NOTE: Installing to /data/local/tmp (Android, no Termux detected)."
    echo "      This location survives reboots but may be cleared by the system."
    echo "      For a permanent install, use Termux and re-run this script."
    echo ""
fi

# ── Download ──────────────────────────────────────────────────────────────────
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL --progress-bar -o "${TMP_DIR}/${ARCHIVE}" "${URL}"
elif command -v wget >/dev/null 2>&1; then
    wget -q --show-progress -O "${TMP_DIR}/${ARCHIVE}" "${URL}"
else
    echo "ERROR: neither curl nor wget found"
    exit 1
fi

# ── Extract ───────────────────────────────────────────────────────────────────
echo "Extracting..."
cd "$TMP_DIR"
if [ "$PLATFORM" = "windows" ]; then
    unzip -q "${ARCHIVE}"
else
    tar xzf "${ARCHIVE}"
fi

# ── Install ───────────────────────────────────────────────────────────────────
chmod +x "${BINARY}"
mkdir -p "${INSTALL_DIR}"
cp "${BINARY}" "${INSTALL_PATH}"

echo ""
echo "✓ Installed: ${INSTALL_PATH}"
echo ""

# ── Verify ────────────────────────────────────────────────────────────────────
if "${INSTALL_PATH}" -v 2>/dev/null; then
    echo ""
    echo "✓ go-tagger is working."
else
    echo "Binary installed. Run: ${INSTALL_PATH} -v"
fi

# ── PATH hint ─────────────────────────────────────────────────────────────────
case "$INSTALL_DIR" in
    /data/local/tmp)
        echo ""
        echo "Add to shell config for convenience:"
        echo "  echo 'alias go-tagger=${INSTALL_PATH}' >> ~/.zshrc"
        echo "  echo 'alias go-tagger=${INSTALL_PATH}' >> ~/.bashrc"
        ;;
    "$HOME/bin"|"$HOME/.local/bin")
        echo ""
        if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
            echo "Add to PATH:"
            echo "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.zshrc"
        fi
        ;;
esac
