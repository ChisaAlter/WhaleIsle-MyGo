# Installed-plugin version updates

Open Marketplace → Installed → Check plugin updates. This fetches bounded public npm metadata only. Each recognized installed row shows current → latest and either Update available, No newer version, or unknown. Click a row's Update to VERSION button to explicitly request that exact version through the existing market install pipeline. Build-script approval, compatibility exemptions, cancellation, log display and result history are reused. No bulk or automatic installation was added.

## Trust and lifecycle

- Only uniquely catalog-matched, installed, removable npm bundles with a recorded npm source are eligible. Git, linked/local, ambiguous identities and missing versions stay unknown.
- The metadata endpoint is registry.npmjs.org/PACKAGE/latest; returned package identity must match. No README parsing or command extraction is involved.
- Comparison supports strict stable X.Y.Z versions only. Prereleases, tags and malformed/overflowing components are unknown, not an update or a claim that the plugin is current.
- Explicit updates pin PACKAGE@VERSION. Receipts bind current/target version, check generation, catalog revision, inventory revision and runtime attachment, and expire after 30 minutes.
- Refreshing catalog/inventory, disconnecting/replacing runtime or starting another version check invalidates selection. Late results are discarded.
- Pending build/exemption approval retains its update receipt; terminal outcomes refresh inventory. A manual refresh during approval deliberately requires checking/selecting again.
- Network checks have an 8-second per-package timeout and a 90-second overall bound. Errors show unknown/retry; checking itself never mutates runtime state.

## Verification

Focused fixtures cover npm identity/endpoint verification, stable numeric comparison, pinned selection, stale inventory rejection, unknown-version rejection and runtime detachment. Windows amd64 compilation validates the full native wiring. No actual plugin install, host command, deployment or user-computer operation was performed.
