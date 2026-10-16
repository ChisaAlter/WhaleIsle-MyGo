#!/bin/sh
# Build only. Signing uses the owner's existing external release workflow.
# No keychain access, certificate creation, installation, or launch occurs here.
set -eu
if [ "$#" -ne 2 ]; then echo "usage: $0 amd64|arm64 NEW_OUTPUT_DIRECTORY" >&2; exit 2; fi
[ "$(uname -s)" = Darwin ] || { echo "macOS SDK and clang are required" >&2; exit 1; }
case "$1" in amd64) arch=x86_64;; arm64) arch=arm64;; *) echo "unsupported architecture" >&2; exit 2;; esac
[ ! -e "$2" ] || { echo "choose a fresh output directory" >&2; exit 1; }
project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$2"
out=$(CDPATH= cd -- "$2" && pwd)
app="$out/WhaleIsle Preview.app"
mkdir -p "$app/Contents/MacOS"
/usr/bin/xcrun --sdk macosx clang -arch "$arch" -mmacosx-version-min=11.0 -fobjc-arc -fblocks \
  -Wall -Wextra -Wno-deprecated-declarations -Wno-unused-parameter \
  -framework Cocoa -framework WebKit -framework Security \
  "$project/native/macos-preview/main.m" -o "$app/Contents/MacOS/whaleisle-preview"
cat > "$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>org.whaleisle.preview</string>
<key>CFBundleExecutable</key><string>whaleisle-preview</string>
<key>CFBundleName</key><string>WhaleIsle Preview</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleVersion</key><string>1.0.0</string>
<key>CFBundleShortVersionString</key><string>1.0.0</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
cp "$project/native/macos-preview/preview.entitlements" "$out/preview.entitlements"
echo "Built UNSIGNED legacy compatibility helper: $app"
echo "Use the existing external signing workflow with preview.entitlements and the host's Apple signing team."
echo "Then provide this signed app as previewHelperBundle to package-macos.mjs stage."
echo "Runtime acceptance on macOS is still required; no helper was launched."
