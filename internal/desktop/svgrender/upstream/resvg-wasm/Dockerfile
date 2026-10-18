# syntax=docker/dockerfile:1
#
# Reproducible build of the resvg -> wasm32-wasip1 module WITH raster-image rendering,
# for the pure-Go wazero runtime (renderer/png_converter.go via the vendored Go wrapper).
#
# Build + extract the artifacts to the host (no local Rust toolchain needed — only Docker):
#   docker build --target artifact -o out .
#   -> out/resvg.wasm   (raw module; gzip + embed into the Go wrapper, see Makefile `install`)
#   -> out/Cargo.lock   (resolved dependency versions — commit this for reproducible/pinned builds)
#
# This same Dockerfile is portable to GitHub Actions later with no changes
# (a `docker build -o` step in a workflow), which is why all build logic lives here, not in
# host scripts.

ARG RUST_VERSION=1.88
FROM rust:${RUST_VERSION}-slim-bookworm AS build
# Supply-chain hardening: create a non-root build user and hand it the toolchain + cargo dirs, so
# EVERYTHING below runs unprivileged — `rustup target add`, the dependency fetch, AND the compile,
# including any crate build.rs / proc-macro code (the actual attack surface). No untrusted dependency
# code ever runs as root; the only root operations are the useradd/chown setup, which run no project
# code. A compromised dependency therefore cannot escalate beyond the unprivileged `builder` user.
RUN useradd --create-home --uid 10001 builder \
    && chown -R builder:builder /usr/local/rustup /usr/local/cargo
USER builder
WORKDIR /home/builder/build
RUN rustup target add wasm32-wasip1
# Enable WASM SIMD (128-bit vectors): wazero executes simd128, and it lets LLVM/tiny-skia vectorise the
# pixel-pushing hot loops. Combined with opt-level=3 this closes most of the gap to the V8-run JS module.
ENV RUSTFLAGS="-C target-feature=+simd128"
# Manifest first (so Docker caches the slow fetch+compile layer). Cargo.lock is optional on the first
# build (cargo resolves + writes one); once committed it pins exact dep versions. Pairing Cargo.lock*
# with the always-present Cargo.toml in one COPY lets the wildcard match zero files without erroring.
# `--locked` below is enforced only when a lock is actually present.
COPY --chown=builder:builder Cargo.toml Cargo.lock* ./
COPY --chown=builder:builder wasm ./wasm
RUN if [ -f Cargo.lock ]; then \
        cargo build --release --target wasm32-wasip1 --locked; \
    else \
        cargo build --release --target wasm32-wasip1; \
    fi
# wazero JIT-recompiles the WASM to native itself, so a binaryen wasm-opt post-pass is redundant (it even
# regressed slightly in benchmarking) — we rely on LLVM (opt-level=3 + simd128 + fat-LTO/codegen-units=1).
RUN cp target/wasm32-wasip1/release/resvg.wasm /home/builder/resvg.wasm

# Export ONLY the artifacts (scratch has no layers) so `docker build -o out` drops them on the host.
FROM scratch AS artifact
# This image only exports build artifacts via `docker build --target artifact -o out`; it is never run
# as a container. Declare explicitly that it has no healthcheck — the honest statement for an
# artifact-export image, which also satisfies the scanner without an ignore/suppression.
HEALTHCHECK NONE
COPY --from=build /home/builder/resvg.wasm /resvg.wasm
COPY --from=build /home/builder/build/Cargo.lock /Cargo.lock
