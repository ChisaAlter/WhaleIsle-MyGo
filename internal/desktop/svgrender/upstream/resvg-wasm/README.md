# resvg-wasm — our own resvg → WASM build (with images)

This directory builds the resvg rasterizer to a `wasm32-wasip1` module that the Go port runs on
the pure-Go [wazero](https://github.com/tetratelabs/wazero) runtime — **the same engine the JS
reference uses**, just compiled for WASI instead of wasm-bindgen.

## Why this exists

The JS reference (`@resvg/resvg-wasm@2.6.2`) renders embedded `<image>` elements natively, so it
needs no image compositor. The Go port used `github.com/kanrichan/resvg-go@v0.0.1`, whose WASM was
built with `default-features = false, features = ["text"]` — i.e. **raster-image rendering stripped
out** to shrink the file. That is the only reason the Go port originally had to composite images
itself, and that in-process compositor (since removed) could only approximate crop / tile / rotate.

There is **no official or well-maintained `wasm32-wasi` resvg build with images** to depend on (the
resvg project ships an official *wasm-bindgen* build for JS, but wazero runs *WASI* — different
dialects, not interchangeable). So we build our own, the same way upstream does, with
`raster-images` turned back on. Result: resvg draws images itself → **bit-identical to the JS
reference**, the compositor is deleted, and the binary stays CGO-free (the `.wasm` runs on wazero;
no native `.so` or CGo binding).

## Build (Docker only — no local Rust toolchain)

```bash
make install     # build the .wasm in Docker, then gzip + embed it into the Go wrapper
# or step by step:
make wasm         # -> out/resvg.wasm + out/Cargo.lock
make lock         # copy the resolved Cargo.lock next to Cargo.toml (commit it to pin deps)
```

Everything runs inside the `rust:${RUST_VERSION}` container defined in the `Dockerfile`; the host
only needs Docker.

### CI rebuild-verify

`internal/resvg/wasm/resvg.wasm.gz` is a **generated artifact**, committed because `go get` cannot run
a build step. To keep it honest, the `resvg-wasm-verify` CI job rebuilds the module from this directory
in Docker and runs the renderer's resvg tests against the result — proving the source compiles to a
working resvg of the pinned version, never a hand-uploaded mystery. (The `.wasm` is platform-independent
and cross-environment builds aren't byte-identical, so this is a *functional* check; tamper-protection of
the committed bytes is `internal/resvg/version_test.go`'s SHA pin.) The `Dockerfile` is hermetic (public
`rust:*-slim-bookworm` base, `COPY` only from the build context, `cargo build` — **no host paths,
mounts, or local scripts**); the artifact is extracted via `docker build -o`, and `make install`'s only
non-Docker step is `gzip -9 -n`. The verify job is essentially:

```yaml
# from the resvg-wasm-verify job in .github/workflows/ci.yml
- uses: docker/setup-buildx-action@<sha>
- run: cd resvg-wasm && make install                           # rebuild + gzip into internal/resvg
- run: go test ./internal/resvg/ -run 'TestRender...' -count=1  # the rebuilt module must render correctly
```

The `internal/resvg/version_test.go` gate makes a rebuilt-but-not-re-pinned wasm fail CI, so the
committed artifact and the pinned resvg version can't silently drift.

## Version pinning & reproducibility

- **resvg version** is pinned in `Cargo.toml` and locked in `Cargo.lock` (commit the lock after the
  first `make lock`). The pin is **resvg 0.46** — the version that renders pixel-identical to the JS
  reference (`@resvg/resvg-wasm@2.6.2`'s lineage) and is the fastest across the 0.35→0.47 range;
  `Cargo.lock` locks the full dependency set.
- **Rust toolchain** is pinned via `RUST_VERSION` (Docker base image tag).
- **Reproducible gzip**: `make install` uses `gzip -n` (drops name+mtime) so the embedded artifact
  is byte-stable.

## Version detection (drift guard)

The Docker build is **byte-deterministic** — verified 2026-06-25: a `--no-cache` full recompile
produced a byte-identical `.wasm` (1655709 bytes both times). So version-detection is a simple
checksum + version assertion: record the resvg version (from `Cargo.lock`) and the **SHA-256 of the
built `.wasm`** next to the Go embed, and a Go test asserts the embedded module matches both — failing
CI if anyone rebuilds against a different resvg version (or a drifted toolchain) without updating the
pin. No embedded version string needed — `internal/resvg/version_test.go` is the active drift guard.

## Bumping resvg

1. Edit the `resvg`/`usvg` versions in `Cargo.toml`.
2. `make lock` (regenerate `Cargo.lock`), then `make install` (rebuild + re-embed).
3. Update the recorded expected version in the Go version-detection test.
4. Re-run the renderer's image VRT against the JS goldens to confirm parity.

## Files

| File | Purpose |
|------|---------|
| `Cargo.toml` | pinned deps; `raster-images` re-enabled |
| `wasm/resvg.rs` | the C-ABI shim (clean-room, authored over resvg's MPL-2.0 API; **not** derived from the GPL-3.0 `kanrichan/resvg-go`) |
| `Dockerfile` | reproducible `wasm32-wasip1` build + artifact export |
| `Makefile` | `wasm` / `install` / `lock` / `clean` |
