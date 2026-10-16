# Build the resvg WASM (with raster images) reproducibly in Docker and install it into the
# vendored Go wrapper. No local Rust toolchain required — only Docker.
#
#   make wasm      # build out/resvg.wasm + out/Cargo.lock via Docker
#   make install   # build, then gzip + place the module where the Go wrapper embeds it
#   make lock      # copy the resolved Cargo.lock back next to Cargo.toml (pin deps)
#   make clean

# resvg 0.47's dep tree (polycool 0.4.0 etc.) uses Rust edition 2024 → needs Rust ≥ 1.85; pin 1.88.
RUST_VERSION ?= 1.88
OUT          := out
# Where the vendored Go wrapper embeds the module (//go:embed).
GO_EMBED     := ../internal/resvg/wasm/resvg.wasm.gz

.PHONY: wasm install lock native clean

# Host-native static lib for the OPTIONAL CGo backend (internal/resvg/resvg_cgo.go, -tags cgo_resvg).
# Reuses the SAME shim source (wasm/resvg.rs) as a `staticlib`. Uses the LOCAL Rust toolchain (unlike the
# Docker wasm build), targeting the host arch → native/target/release/libresvg_native.a, which the CGo
# file links. The default (non-cgo) build never needs this. Cross-compiling it for the target
# architectures (linux/arm64, linux/amd64) requires a cross Rust target + a matching CGo cross-linker.
native:
	cargo build --release --manifest-path native/Cargo.toml
	@echo "built native/target/release/libresvg_native.a (host arch; link with -tags cgo_resvg)"
wasm:
	docker build --target artifact --build-arg RUST_VERSION=$(RUST_VERSION) -o $(OUT) .
	@echo "built $(OUT)/resvg.wasm ($$(wc -c < $(OUT)/resvg.wasm | tr -d ' ') bytes)"

# Reproducible gzip: -n drops the name+mtime so the embedded artifact is byte-stable across builds.
install: wasm
	@mkdir -p $(dir $(GO_EMBED))
	gzip -9 -n -c $(OUT)/resvg.wasm > $(GO_EMBED)
	@echo "installed -> $(GO_EMBED)"

lock: wasm
	cp $(OUT)/Cargo.lock ./Cargo.lock
	@echo "pinned deps -> ./Cargo.lock (commit this)"

clean:
	rm -rf $(OUT)
