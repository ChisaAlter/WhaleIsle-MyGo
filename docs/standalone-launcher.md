# Standalone native launcher (deprecated)

**Status: deprecated and no longer distributed.** The product ships as a single
full installer (`WhaleIsle-MyGo-Setup-<version>.exe`) that already bundles the Host
runtime. Version listing, download (pause/resume), checksum verification,
installer hand-off, imports, components and plugin recovery all live in the
desktop app's built-in start page (`internal/ui/launcher` +
`cmd/whaleisle/native_releases.go`).

Why it was dropped:

- It duplicated the desktop's update-center state machine, so every download or
  install fix had to be made twice.
- It was never part of the installer or the GitHub release assets, so end users
  could not obtain it.
- The "cannot download" failure it was meant to address was a catalog problem
  (v0.1.0 shipped without the runtime), fixed by the full v0.1.1 package, not an
  architecture problem.

`cmd/whaleisle-launcher` and `internal/desktop/externallauncher` remain in the
tree as reference code only. They are not built by CI, packaged, or covered by
`scripts/verify-inspector-targets.ps1`. If a small bootstrap download is wanted
later, build a single-purpose web installer (download, verify, install, exit)
instead of reviving this resident manager.