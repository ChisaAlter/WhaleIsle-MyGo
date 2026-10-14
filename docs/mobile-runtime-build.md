# Mobile runtime build and platform boundary

The mobile runtime is the original ChisaCode server/client plus MyGo's existing
desktop shell hooks. Building it does not pair a phone, start the daemon, open a
tunnel, configure credentials, or run a model.

`scripts/build-mobile-runtime.ps1` remains the Windows entrypoint. It requires an
official Windows Node distribution and produces Windows native modules. Its
browser step now calls `mobile-runtime.mjs bundle`, which uses esbuild's public JS
API. The original script's `node esbuild/bin/esbuild` invocation fails when npm
has installed a native ELF/Mach-O executable at that path.

For a Linux/macOS build-only workflow, use the same prepared source in an isolated
work directory and an official same-platform Node distribution with headers:

1. `mobile-runtime.mjs prepare SOURCE WORK VERIFIED_SILERO_FILE`.
2. In WORK/vendor/chisacode-remote: `npm ci --ignore-scripts --no-audit --no-fund`
   against the official registry, then `npm rebuild esbuild better-sqlite3 node-pty`.
3. Run the original `npm run build:server` (server dependencies, server, CLI).
4. Run `mobile-runtime.mjs bundle WORK` and `mobile-runtime.mjs record WORK`.
   The latter only loads native modules and runs SQLite `select 1` in memory.
5. `mobile-runtime.mjs stage WORK BUILT_HARNESS NODE_DISTRIBUTION DESTINATION`.
6. `mobile-runtime.mjs verify DESTINATION` checks hashes and dependency links.

Use workspace-local npm cache/TMPDIR and npm_config_nodedir pointing to that
complete Node distribution when development headers are not otherwise installed.
Never replace the source snapshot or alter its lockfile to mask a build failure.

## Source closure

The restored customized snapshot omitted server/src/share/QRCode.ts. Preparation
supplements only that missing original file from the vendored, licensed copy under
scripts/mobile-runtime-source. The importing pairing-qr.ts must match the fixed
90465b67 source exactly, and the supplement has its own SHA256. Existing customized
QRCode.ts files are never replaced. source-provenance.json records the supplement.
Silero remains pinned and validated separately; it is not automatically downloaded.

## Staging

Format-2 manifests record platform/architecture, file hashes and dependency links.
The exact Node executable hash must match the build record. Workspace and pnpm
links are preserved instead of recursively duplicating dependency graphs. External
or changed links fail verification; Windows absolute junctions are rebased after
the staging directory moves. The unrelated Harness CLI test-profile directory is
excluded because it contains an external snapshot fixture link, not runtime code.
The twelve built Harness packages required by ChisaCode's original provider are
linked at its expected node_modules/@deepseek-ai scope without rewriting them.

Format-1 Windows manifests remain readable. Windows package.ps1 still requires
node/node.exe and the Windows mobile runtime. Linux artifacts use node/bin/node;
they cannot satisfy or masquerade as the Windows deployment package. A successful
build/stage does not establish live phone/relay interoperability or runtime UI
acceptance; those require separately authorized use of the matching platform.
