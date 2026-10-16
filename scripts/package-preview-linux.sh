#!/bin/sh
# Build the trusted Linux companion and package an explicitly supplied minimal
# runtime root. This never copies host /usr, /etc, home or display sockets.
set -eu
if [ "$#" -ne 2 ]; then
  echo "usage: $0 TRUSTED_RUNTIME_ROOT OUTPUT_DIRECTORY" >&2
  exit 2
fi
root=$(cd "$1" && pwd)
mkdir -p "$2"
out=$(cd "$2" && pwd)
case "$out/" in "$root/"*) echo "output must be outside the runtime root" >&2; exit 1;; esac
case "$root" in /|/usr|/etc|/home|/tmp|/run) echo "refusing a host system directory as runtime" >&2; exit 1;; esac
for bin in usr/bin/Xvfb usr/bin/dbus-run-session usr/bin/xdotool; do
  [ -x "$root/$bin" ] || { echo "runtime missing $bin" >&2; exit 1; }
done
[ ! -e "$out/whaleisle-preview.runtime" ] || { echo "runtime destination already exists; choose a new output directory" >&2; exit 1; }
# Runtime must be curated as an application asset: only system libraries,
# WebKit subprocesses, fonts, Xvfb and a private session bus, no user data.
CGO_ENABLED=0 GOOS=linux go build -trimpath -o "$out/whaleisle-preview" ./cmd/whaleisle-preview
mkdir "$out/whaleisle-preview.runtime"
cp -a "$root/." "$out/whaleisle-preview.runtime/"
echo "Packaged Linux capture companion in $out. GUI/namespace runtime acceptance is still required."
