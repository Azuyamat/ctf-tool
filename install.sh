#!/bin/sh
set -eu

REPO="Azuyamat/ctf-tool"
VERSION="${CTF_VERSION:-latest}"
INSTALL_DIR="${CTF_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	echo "error: $*" >&2
	exit 1
}

download() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$2" "$1"
	else
		fail "curl or wget is required"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required"
	fi
}

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported OS: $(uname -s)" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "unsupported architecture: $(uname -m)" ;;
esac

asset="ctf_${os}_${arch}.tar.gz"
if [ "$VERSION" = latest ]; then
	base="https://github.com/$REPO/releases/latest/download"
else
	base="https://github.com/$REPO/releases/download/$VERSION"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $asset ($VERSION)"
download "$base/$asset" "$tmp/$asset" || fail "could not download $base/$asset"
download "$base/checksums.txt" "$tmp/checksums.txt" || fail "could not download checksums"

expected=$(awk -v asset="$asset" '$2 == asset { print $1 }' "$tmp/checksums.txt")
[ -n "$expected" ] || fail "no checksum listed for $asset"
[ "$(sha256 "$tmp/$asset")" = "$expected" ] || fail "checksum mismatch for $asset"

tar -xzf "$tmp/$asset" -C "$tmp" ctf
mkdir -p "$INSTALL_DIR"
install -m 755 "$tmp/ctf" "$INSTALL_DIR/ctf"
echo "Installed ctf to $INSTALL_DIR/ctf"

case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*) echo "Add $INSTALL_DIR to your PATH, e.g. in ~/.zshrc: export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
