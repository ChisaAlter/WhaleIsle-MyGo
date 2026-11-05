# AVIF native image codec provenance

The unmodified `avif.wasm.gz`, C ABI source and build recipes are from
`github.com/gen2brain/avif` v0.6.0, commit
`1fd2a9a51895ca03a192dd49917840794c8a77a1`, downloaded from the official Go module
registry at https://proxy.golang.org/github.com/gen2brain/avif/@v/v0.6.0.zip.

- Compressed WASM: 1,292,476 bytes; decompressed WASM: 3,462,314 bytes.
- Compressed SHA-256: `92ffd604d276359c2df829effc615cd9f286c36ece5761644929e4396eb97360`.
- Upstream: https://github.com/gen2brain/avif/tree/v0.6.0.
- The upstream recipe identifies libavif v1.4.2, dav1d 1.5.3, aom v3.14.1 and
  libyuv stable. Its libyuv source is not pinned to a commit; we preserve the exact
  published binary rather than claim a reproducible dependency rebuild.
- Wrapper/fixtures/orientation helpers: MIT (`LICENSE.gen2brain`).
- libavif and dav1d: BSD-2-Clause; aom and libyuv: BSD-3-Clause.
  All included notices are retained alongside the binary. The libavif notice
  also contains upstream source-specific third-party notices.
- Additional upstream notices: https://raw.githubusercontent.com/videolan/dav1d/1.5.3/COPYING
  and https://chromium.googlesource.com/libyuv/libyuv/+/refs/heads/stable/LICENSE.

The application embeds the compressed binary and runs only its decoder ABI in
wazero v1.12.0. There is no dynamic library lookup, cgo, subprocess, filesystem
preopen, environment inheritance or network capability. Each operation owns a
fresh WASM memory instance; returned pixels are copied before it closes. Runtime
memory is capped at 512 MiB and pixel decoding has a ten-second context deadline.

The application retains its existing 32 MiB input / 32 million pixel policy and
single-primary-item still-image subset. It supports 8-, 10- and 12-bit pixels and
applies irot/imir orientation. The still decoder continues to reject animated
AVIF, auxiliary/alpha items, grids and collections rather than flatten them.
Composer and Market separately use the existing `decode_all` ABI for bounded
AVIS sequences: one AV1 color track with an optional matching alpha track,
sample timing and edit-list repetition, at most 256 frames / 32 million retained
canvas pixels. The Go sequence gate checks sample extents and dimensions before
WASM output allocation. Duration sentinels detect incomplete output because the
upstream C ABI reports success even after `avifDecoderNextImage` fails. No C or
WASM binary changed; Files retains its existing still-only AVIF path. Unsupported
sequence properties (including clean-aperture crop, layered images and additional
color tracks) return explicit errors. This is native image decoding, not browser
capture. HDR tone mapping and ICC color management are not added.

`../avif_orientation.go` is adapted only by changing the package name from
upstream `orientation.go`. Fixtures in `../testdata/test_rot.avif`, `test10.avif`
and `test.avifs` are copied unchanged from this same module version.
