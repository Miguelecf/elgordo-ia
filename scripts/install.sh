#!/bin/sh
set -eu

REPO="Miguelecf/elgordo-ia"
RUNTIME_DIR="${ELGORDO_RUNTIME_DIR:-$HOME/.config/elgordo/runtime}"
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t elgordo)"
staged_binary=""
previous_binary=""
new_binary_installed=0
installation_complete=0

cleanup() {
  if [ "$installation_complete" -ne 1 ]; then
    if [ "$new_binary_installed" -eq 1 ]; then
      rm -f "$RUNTIME_DIR/elgordo"
    fi
    if [ -n "$previous_binary" ] && [ -e "$previous_binary" ]; then
      mv "$previous_binary" "$RUNTIME_DIR/elgordo"
    fi
    if [ -n "$staged_binary" ]; then
      rm -f "$staged_binary"
    fi
  fi
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

if ! command -v curl >/dev/null 2>&1; then
  echo "error: curl is required to install ElGordo" >&2
  exit 1
fi

if [ ! -r /dev/tty ]; then
  echo "error: installation requires an interactive terminal for explicit dependency consent" >&2
  echo "Download the release archive manually when running without a TTY." >&2
  exit 1
fi

os="$(uname -s)"
case "$os" in
  Darwin) goos="darwin" ;;
  Linux) goos="linux" ;;
  *)
    echo "error: unsupported OS $os; v0.1.0 supports macOS, Linux, and WSL" >&2
    exit 1
    ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *)
    echo "error: unsupported architecture $arch" >&2
    exit 1
    ;;
esac

if [ -n "${ELGORDO_VERSION:-}" ]; then
  version="$ELGORDO_VERSION"
else
  version="$(curl --retry 3 --max-time 120 -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | sed -n '1p')"
fi

if [ -z "$version" ]; then
  echo "error: no ElGordo release found" >&2
  exit 1
fi

plain_version="${version#v}"
archive="elgordo_${plain_version}_${goos}_${goarch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/$version"

echo "Downloading ElGordo $version for $goos/$goarch..."
curl --retry 3 --max-time 120 -fsSL "$base_url/$archive" -o "$TMP_DIR/$archive"
curl --retry 3 --max-time 120 -fsSL "$base_url/checksums.txt" -o "$TMP_DIR/checksums.txt"

expected="$(grep " $archive$" "$TMP_DIR/checksums.txt" | sed 's/[[:space:]].*//')"
if [ -z "$expected" ]; then
  echo "error: release checksum for $archive was not found" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$TMP_DIR/$archive" | sed 's/[[:space:]].*//')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$TMP_DIR/$archive" | sed 's/[[:space:]].*//')"
else
  echo "error: sha256sum or shasum is required" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "error: checksum verification failed" >&2
  exit 1
fi

if ! tar -tzf "$TMP_DIR/$archive" | grep -qx "elgordo"; then
  echo "error: release archive does not contain the expected elgordo binary" >&2
  exit 1
fi
tar -xzf "$TMP_DIR/$archive" -C "$TMP_DIR" "elgordo"
chmod +x "$TMP_DIR/elgordo"

mkdir -p "$RUNTIME_DIR"
staged_binary="$RUNTIME_DIR/.elgordo.new.$$"
previous_binary="$RUNTIME_DIR/.elgordo.previous.$$"
cp "$TMP_DIR/elgordo" "$staged_binary"
chmod +x "$staged_binary"
if [ -e "$RUNTIME_DIR/elgordo" ]; then
  mv "$RUNTIME_DIR/elgordo" "$previous_binary"
fi
mv "$staged_binary" "$RUNTIME_DIR/elgordo"
new_binary_installed=1

if ! "$RUNTIME_DIR/elgordo" install </dev/tty; then
  echo "error: ElGordo setup failed; the previous binary was restored" >&2
  exit 1
fi
rm -f "$previous_binary"
installation_complete=1

echo "Installed ElGordo runtime to $RUNTIME_DIR/elgordo"
echo "OpenCode is ready. Start opencode; elgordo-ia is available as the primary agent."
