# Original mobile runtime build/staging

The Go phone bridge requires an actual built original DshdRemote runtime, not
just `src/main/dshd-remote.js`. Its packaged root is
`runtime/vendor/dshd-remote/`; the existing component override expects that same
layout at `components/dshd-remote/versions/<version>/` with `installed.json`.
These scripts do not install/activate a component or write userData. The generic
component install action remains unsupported; do not advertise an online installer.

## Sources and provenance

Set `WHALEISLE_ELECTRON_SOURCE` to a complete original Electron source snapshot
(default: `reference/electron`) and `WHALEISLE_NODE_DIR` to an official Windows
Node distribution including npm. Use the same Node for building and shipping.
The supplied snapshot identifies Electron 0.3.3 and ChisaCode 1.0.3
(`vendor/chisacode-remote/package.json`, repository
https://github.com/ChisaAlter/ChisaCode). The separate harness pin is
`dsh-v0.2.1-alpha.1`, SHA `5badb15009ae1756c3afe0ae0cef1faafc290ccc`.
Do not mistake that harness SHA for a ChisaCode commit. ChisaCode's vendored fork
is identified by its source snapshot and recorded lockfile/manager SHA256, not an
invented upstream commit. Retain its LICENSE, NOTICE and corresponding source
when distributing (AGPL-3.0-or-later).

Run `scripts/build-harness.ps1` then `scripts/package.ps1`.
The former copies missing harness/plugin source directories, builds the harness,
then invokes `build-mobile-runtime.ps1`. To change source snapshots, first use a
fresh runtime build/vendor workspace rather than mixing old and new directories.

The phone build uses a fresh workspace, `npm ci --ignore-scripts`, explicit
`npm rebuild esbuild better-sqlite3 node-pty`, original `npm run build:server`,
and original `bundle-chisacode-mobile-client.mjs`. It deliberately never invokes
`prepare-dshd-remote.mjs --runtime`: that script targets Electron's SQLite ABI
and starts a daemon probe. The MyGo script checks native module loading using
plain Node and an in-memory SQLite database, without starting a daemon, opening
a socket or generating credentials. Dependency retrieval/build occurs only when
a developer actually runs the build script.

Staging preserves the original module/mobile/vendor topology, dereferences
workspace links, includes the complete built harness needed by the manager's
fallback, and ships the exact Node executable whose ABI passed the native check.
It currently preserves the locked workspace dependencies rather than attempting
an unverified production prune, so output can be large. A SHA256 inventory covers
all staged files. Packaging verifies the source and copied inventories before
writing `.runtime-staged`; missing or changed resources fail the package build.
`WHALEISLE_PACKAGE_SKIP_RUNTIME=1` explicitly creates a shell-only package.

## Verified limits in the supplied 2026-10-13 snapshot

The original server build script requires this source asset, which is absent
from both inspected source/baseline trees:

`vendor/chisacode-remote/packages/server/src/server/speech/providers/local/sherpa/assets/silero_vad.onnx`

Source preparation fails with that exact path before any npm install. Supply the
complete matching original source release including this asset; these scripts do
not download a substitute, create a dummy model, or bypass the original build.
Additional missing assets/dependencies may still be surfaced by the real build.

Validation performed here: one temporary-directory fixture exercising staging,
its complete required inventory, repeated staging, missing bundle/manager
failures, and missing source-model failure. No npm install, Windows native build,
PowerShell execution, daemon startup, pairing credentials or live relay/phone
acceptance was performed. The fixture uses inert files and proves packaging
control flow, not that a working native runtime exists. This is a script-only
change; no additional Go/Windows compilation is claimed or needed for these files.
