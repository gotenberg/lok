#!/bin/sh
# Installs the lok CLI from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/gotenberg/lok/main/install.sh | sh
#
# Environment:
#   LOK_VERSION      release tag to install, for example v1.2.3 (default: latest)
#   LOK_INSTALL_DIR  directory for the binary (default: /usr/local/bin)
#
# Supports Linux on amd64, 386, arm64, armv7 and ppc64le, the platforms
# Gotenberg ships. The binary needs glibc 2.34+ and LibreOffice at runtime.

set -eu

REPO="gotenberg/lok"
RELEASES="https://github.com/${REPO}/releases"
INSTALL_DIR="${LOK_INSTALL_DIR:-/usr/local/bin}"

fail() {
  echo "lok install: $*" >&2
  exit 1
}

need() {
  command -v "$1" > /dev/null 2>&1 || fail "$1 is required"
}

need curl
need tar
need uname

case "$(uname -s)" in
  Linux) os=linux ;;
  *) fail "unsupported OS $(uname -s); lok only runs on Linux" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  i386 | i686) arch=386 ;;
  arm64 | aarch64) arch=arm64 ;;
  armv7l | armv8l) arch=armv7 ;;
  ppc64le) arch=ppc64le ;;
  *) fail "unsupported architecture $(uname -m); see ${RELEASES}" ;;
esac

if command -v sha256sum > /dev/null 2>&1; then
  sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum > /dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  fail "sha256sum or shasum is required"
fi

# The latest-release page redirects to the tag, which avoids the rate-limited
# GitHub API.
tag="${LOK_VERSION:-}"
if [ -z "$tag" ]; then
  tag="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "${RELEASES}/latest")" \
    || fail "no published release found at ${RELEASES}"
  tag="${tag##*/}"
  [ -n "$tag" ] && [ "$tag" != "latest" ] || fail "no published release found"
fi

archive="lok_${tag#v}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading lok ${tag} for ${os}/${arch}"
curl -fsSL -o "${tmp}/${archive}" "${RELEASES}/download/${tag}/${archive}" \
  || fail "cannot download ${archive} from release ${tag}"
curl -fsSL -o "${tmp}/checksums.txt" "${RELEASES}/download/${tag}/checksums.txt" \
  || fail "cannot download checksums.txt from release ${tag}"

want="$(awk -v f="$archive" '$2 == f { print $1 }' "${tmp}/checksums.txt")"
[ -n "$want" ] || fail "${archive} is missing from checksums.txt"
[ "$(sha256 "${tmp}/${archive}")" = "$want" ] || fail "checksum mismatch for ${archive}"

tar -xzf "${tmp}/${archive}" -C "$tmp" lok

if [ -w "$INSTALL_DIR" ] || { [ ! -e "$INSTALL_DIR" ] && mkdir -p "$INSTALL_DIR" 2> /dev/null; }; then
  sudo=""
else
  need sudo
  sudo="sudo"
  echo "${INSTALL_DIR} is not writable, using sudo"
fi
$sudo mkdir -p "$INSTALL_DIR"
$sudo install -m 0755 "${tmp}/lok" "${INSTALL_DIR}/lok"

echo "Installed $("${INSTALL_DIR}/lok" --version) to ${INSTALL_DIR}/lok"
