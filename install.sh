#!/bin/sh
# Installs the POEditor CLI on macOS or Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/POEditor/poeditor-cli/master/install.sh | sh
#
# Environment:
#   POEDITOR_VERSION       release to install, e.g. v0.1.0 (default: the latest release)
#   POEDITOR_INSTALL_DIR   where to put the binary (default: /usr/local/bin if writable,
#                          otherwise ~/.local/bin)
set -eu

repo="https://github.com/POEditor/poeditor-cli"
base="${POEDITOR_RELEASES_URL:-$repo/releases}" # overridden only by tests

fail() {
	echo "poeditor install: $*" >&2
	exit 1
}

# fetch URL FILE downloads URL into FILE with curl or wget.
fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		fail "needs curl or wget"
	fi
}

# latest_tag prints the newest release tag, read from where /releases/latest redirects to.
latest_tag() {
	if command -v curl >/dev/null 2>&1; then
		url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$base/latest")
	else
		url=$(wget -q -S --spider "$base/latest" 2>&1 | sed -n 's/^ *[Ll]ocation: *//p' | tail -n 1)
	fi
	tag=${url##*/}
	case "$tag" in
	v[0-9]*) echo "$tag" ;;
	*) fail "could not find the latest release at $base/latest" ;;
	esac
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "needs sha256sum or shasum to verify the download"
	fi
}

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported OS $(uname -s); on Windows use install.ps1" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "unsupported architecture $(uname -m)" ;;
esac

tag="${POEDITOR_VERSION:-$(latest_tag)}"
case "$tag" in
v*) ;;
*) tag="v$tag" ;;
esac
archive="poeditor_${tag#v}_${os}_${arch}.tar.gz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading poeditor $tag for $os/$arch..."
fetch "$base/download/$tag/$archive" "$tmp/$archive" || fail "could not download $archive"
fetch "$base/download/$tag/checksums.txt" "$tmp/checksums.txt" || fail "could not download checksums.txt"

want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || fail "$archive is not listed in checksums.txt"
[ "$(sha256 "$tmp/$archive")" = "$want" ] || fail "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" poeditor

if [ -n "${POEDITOR_INSTALL_DIR:-}" ]; then
	dir=$POEDITOR_INSTALL_DIR
elif [ -w /usr/local/bin ]; then
	dir=/usr/local/bin
else
	dir="$HOME/.local/bin"
fi
mkdir -p "$dir"
# Write next to the target and rename, so a running poeditor is never half-overwritten.
cp "$tmp/poeditor" "$dir/.poeditor.new"
chmod 755 "$dir/.poeditor.new"
mv -f "$dir/.poeditor.new" "$dir/poeditor"

echo "Installed $("$dir/poeditor" --version) to $dir/poeditor"
case ":$PATH:" in
*":$dir:"*) ;;
*) echo "Note: $dir is not on your PATH; add it to run poeditor by name." ;;
esac
