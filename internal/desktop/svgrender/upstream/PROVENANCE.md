# Vendored SVG renderer provenance

- Source: github.com/emulatorchen/pptx-glimpse-go v0.0.2
- Commit: e4e39adaf74923525e02b7840bbf38f239cc5a5a
- Go proxy module checksum: h1:+fIRae5Qu6H7PNNIxTVfp5N7Gfp57UWgp4ZsbrcJ768=
- Original wrapper: internal/resvg/resvg.go (MIT; LICENSE and NOTICE retained).
- Original artifact: internal/resvg/wasm/resvg.wasm.gz
- Artifact SHA-256: c9dc354506393777df9bce99fc6143786f5cc6e990031cfd371dc4bb3f6eb78d
- Pinned resvg source version: 0.46.0 (see retained Cargo.toml/Cargo.lock).
- Source and build recipe: resvg-wasm/ in this directory. No CGo/native backend copied.
- Local wrapper changes: package name, no runtime generation directive, bounded WASM memory,
  context cancellation, bounded PNG output, explicit trusted font bytes and no WASI filesystem.
- Runtime: github.com/tetratelabs/wazero v1.12.0 (Apache-2.0).

Only the verified Go registry artifact was downloaded; no build script, WASM,
renderer, native binary or test was executed during this source integration.
The embedded module is renderer implementation code, never user-supplied WASM.

Pinned resvg v0.46.0 is dual MIT/Apache-2.0, verified against the tag's
LICENSE-MIT and LICENSE-APACHE (both included), despite outdated MPL wording
in the upstream wrapper comment. Source is available at
https://github.com/linebender/resvg/tree/v0.46.0 and the precise dependency
versions/checksums are retained in upstream/resvg-wasm/Cargo.lock.
