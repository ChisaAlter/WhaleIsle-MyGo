# Explicit current Host source selection

The optional current Host builder supports two exact upstream commits:

- `90465b67ec16765631d54c195af19d9ff3c3a770` (unchanged default)
- `e9f584be6d3b491ce777bb6823f30bc2bbc6f195` (explicit opt-in)

For e9, use a clean Git checkout of ChisaAlter/WhaleIsle at that exact commit:

```powershell
./scripts/build-current-host.ps1 -SourceRoot C:\sources\WhaleIsle-e9 -SourceCommit e9f584be6d3b491ce777bb6823f30bc2bbc6f195 -UnifiedShortcuts -BuildOnly
```

`-UnifiedShortcuts` remains optional. Source preparation still rejects tracked
changes, an unexpected commit, or a source directory without verifiable Git
history. A materialized source snapshot and successful frontend build alone do
not qualify as a prepared official runtime assembly.

The selected commit is retained in `source-selection.json`, the client build
commit environment, `current-host-build.json`, and the staging result. The build
manifest also records the selected lockfile SHA-256. Without `-BuildOnly`, the
default staging destination is `runtime/vendor-current-<commit-prefix>`; e9 does
not reuse the 904 destination. Existing destinations are rejected, and no
installed runtime is selected automatically. Explicit selection still uses
`WHALEISLE_VENDOR` after the full assembly has been built and verified.

Both pins use the same exact-source visual-readiness patch: their original
`ui-visual-replies/src/tools.ts` has SHA-256
`b8bf5270c9db57134ba448b198236186ccdf8f1b548f217b5c0e2210634c295e`.
The required CLI, web, Project, Team and visual reply artifacts must all exist
and match the manifest, on the selected platform/architecture. Supporting the
e9 provenance does not assert desktop bridge, Project overlay, HTML isolation,
or authenticated Host readiness. Those checks remain independent and fail closed.
