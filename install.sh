#!/bin/sh
# Installs the lok CLI from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/gotenberg/lok/main/install.sh | sh
#
# Environment:
#   LOK_VERSION      release tag to install, for example v1.2.3 (default: the newest release)
#   LOK_INSTALL_DIR  directory for the binary (default: /usr/local/bin)
#   LOK_ARCH         amd64, 386, arm64, armv7 or ppc64le (default: detected)
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

# The architecture is the userland's, which uname reports wrong where a kernel
# runs a foreign userland natively: an arm64 kernel running an armv7 image, or
# an amd64 kernel a 386 one, as a multi-platform Docker build does. dpkg knows
# the userland's architecture; LOK_ARCH states it where dpkg is absent.
arch="${LOK_ARCH:-}"
if [ -z "$arch" ]; then
  if command -v dpkg > /dev/null 2>&1; then
    machine="$(dpkg --print-architecture)"
  else
    machine="$(uname -m)"
  fi
  case "$machine" in
    x86_64 | amd64) arch=amd64 ;;
    i386 | i686) arch=386 ;;
    arm64 | aarch64) arch=arm64 ;;
    armhf | armv7l | armv8l) arch=armv7 ;;
    ppc64le | ppc64el) arch=ppc64le ;;
    *) fail "unsupported architecture ${machine}; see ${RELEASES}" ;;
  esac
fi

if command -v sha256sum > /dev/null 2>&1; then
  sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum > /dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  fail "sha256sum or shasum is required"
fi

# Every release is a pre-release, which the latest-release page skips. The
# releases feed lists them newest first, and avoids the rate-limited GitHub API.
tag="${LOK_VERSION:-}"
if [ -z "$tag" ]; then
  feed="$(curl -fsSL "${RELEASES}.atom")" \
    || fail "cannot read the releases at ${RELEASES}"
  tag="$(printf '%s\n' "$feed" | sed -n 's|.*/releases/tag/\([^"]*\)".*|\1|p' | head -n 1)"
  [ -n "$tag" ] || fail "no published release found at ${RELEASES}"
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

# A binary of the wrong architecture installs fine and fails to run: check it
# runs rather than report what the shell says about it.
version="$("${INSTALL_DIR}/lok" --version)" \
  || fail "${INSTALL_DIR}/lok does not run here: it is built for ${arch}; set LOK_ARCH to this userland's architecture (amd64, 386, arm64, armv7 or ppc64le) and check glibc is 2.34+"
echo "Installed ${version} to ${INSTALL_DIR}/lok"
