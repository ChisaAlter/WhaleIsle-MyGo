# Offline portable Windows candidate

Actual directory: ../windows-portable-candidate. Status: portable-candidate-links-not-restored; runtime_verified=false.

The candidate now contains whaleisle.exe (the already compiled 144c8d3 development binary), fixed ghostty-vt.dll/license/provenance, resources/, the verified nine-file pet-render-assets/, runtime/node/, runtime/vendor/ with Host + Project + desktop packages + dshd-remote, and runtime/target-packages/. No EXE or native binary was executed.

EXE SHA256: 88a2b6960d44db10f9914a015f328e0fc6b8f47f4b7e4c5a59afb441904b6812
Terminal DLL SHA256: 53e8e2dd979702e8f2c48107af19be5a76efc2e88eb7cf1097218e899665da41

## ZIP-safe preparation

The portable directory contains ordinary files, without dependency symlinks. Its candidate-links.json records 17,738 internal links with physical candidate-relative targets. PREPARE-LINKS.cmd invokes only the bundled official Windows Node and restore-offline-links.mjs. On Windows, restoration uses directory junctions and same-volume file hardlinks, both ordinary-user operations on supported local filesystems. It does not request administrator privileges, alter ExecutionPolicy or system security, or start the desktop/Host/mobile/model/account.

The restorer prevalidates every manifest path and target; rejects absolute paths, traversal, alternate streams, reserved names, redirected ancestors, missing physical targets and unexpected existing objects. Existing expected links are accepted idempotently; unexpected files/directories are never overwritten. A local NTFS folder is the documented fallback if the filesystem does not support these link types; elevation is not a workaround. Runtime status stays false after restoration.

431 Windows command wrappers were regenerated for the final runtime/node layout. They contain candidate-relative paths. Mutable configurations are independent files; immutable code/assets use same-filesystem hardlinks while stored in this workspace, which ordinary ZIP packaging reads as normal file content. No production .runtime-staged or Windows verified build manifest was fabricated. Original Linux build manifests remain explicitly named source-linux provenance.

Actual assembly log: ../windows-runtime-preparation/portable.log
Actual config: ../windows-runtime-preparation/portable-config.json
One static fixture: ../windows-runtime-preparation/link-fixture.mjs (valid plan, traversal rejected, existing-file collision rejected without overwrite). The portable's real link plan was statically checked once. No whole-tree hash rescan or Windows execution was performed.

## Code handoff

portable.patch adds package-offline-candidate.mjs and restore-offline-links.mjs, and adds the final Node-relative option to offline-windows-layout.mjs. Apply after the earlier core.patch. It was checked against the current integrated repository. No Go compilation is needed for these scripts.

Usage: node scripts/package-offline-candidate.mjs CONFIG.json

Config keys: source (prepared offline graph), destination (fresh folder), exe/exeSHA256, terminalDir/terminalSHA256, optional resources. Pet assets were then staged through the existing scripts/pet-render-assets.mjs with the fixed local resource directory, verifying all nine original manifest hashes.

This closes offline folder/layout/link-restoration preparation. Real Windows native initialization, Host/desktop integration and actual visual/terminal behavior remain unverified. The development EXE has not been rebuilt with release version/strip flags. Move/archive the folder before restoring junctions; after relocation use a fresh extraction rather than trusting junctions from the old location. Do not add runtime files or their large manifests to Git.
