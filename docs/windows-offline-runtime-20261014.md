# Windows offline dependency-graph assembly

2026-10-14. Actual candidate retained at ../windows-runtime-candidate.

## Result

The Host and mobile Windows x64 dependency graphs have been assembled, rather than stopping at an asset inventory. Final candidate: 298,426 tracked files, 17,738 internal relative links, 431 candidate-relative Windows .cmd wrappers. Its offline-preparation.json says status=prepared and runtime_verified=false. This is not a Windows startup/native-load result.

- host/: guarded fixed 90465b67 Host, original Project and customized desktop packages.
- mobile/: original built mobile JS and embedded fixed Harness graph; correct SQLite12.11.1 ABI137 and original node-pty1.2.0-beta.11.
- node/: official checksum-verified Node24.19.0 Windows x64; mobile also has its own matching node/node.exe.
- target-packages/: 35 exact-version official Windows platform packages, covering different Host/mobile versions separately; each package is connected at its consumer's node_modules so a broad root alias cannot accidentally select the wrong version.

The existing source artifacts remain Linux builds. Their original manifests were renamed to *.source-linux.json inside the candidate; original source trees/manifests were not relabeled or overwritten. Candidate metadata separately includes actual sourceBuilds, source lock/provenance hashes, targetPlatform=win32, targetArch=x64, exact Node SHA256 and each official asset's URL/integrity/SHA256. All target archives were checked against recorded hashes before completion. Host registry integrities match its fixed pnpm lock where entries exist. Some original mobile npm lock entries specify versions without integrity: those retain exact version + independently checked registry integrity and explicitly do not claim a nonexistent root-lock integrity match.

## Static checks actually performed

One complete assembly scan hashed the candidate and checked every remaining .node as PE x86-64, removed ELF/Mach-O and wrong-architecture PE binaries, checked required Host/Project/mobile entry files, and rejected links escaping the candidate. Exact SQLite exported node_register_module_v137 was established in the preceding asset check; node-pty Windows prebuilds are stable Node-API and retain each package's correct original version. POSIX native/system binaries were removed; no imaginary Windows system addon was substituted.

11,919 configuration files were detached into independent inodes. Immutable code/assets use hardlinks to conserve workspace space; every replacement unlinks first. No input inode was edited or chmodded. Original Host's 12 recorded artifact hashes were checked unchanged at completion. Existing Linux mobile/Host source provenance remains preserved.

Generated Linux .bin launchers were removed (including absolute build-machine NODE_PATHs); 431 Windows .cmd wrappers use only candidate-relative paths. Targeted wrapper hashes/config metadata were updated without repeating the whole-tree content scan.

## Script delivery

core.patch adds scripts/offline-windows-runtime.mjs and offline-windows-layout.mjs; both current-host.mjs and mobile-runtime.mjs expose prepare-offline-windows CONFIG.json. Existing native finalize/verify/package gates are unchanged.

Actual invocation:

    node scripts/current-host.mjs prepare-offline-windows CONFIG.json

Example config is ../windows-runtime-preparation/assembly-config.json. Required keys: targetPlatform, targetArch, fresh destination, inputs.host, inputs.mobile, assetsManifest, nodeDir, nodeSHA256, sqliteBinding, sqliteSHA256. assetsManifest entries contain exact name/version, extracted package path, original package.tgz SHA256, source URL and integrity. This recipe deliberately requires the verified Node24.19.0 ABI137 source build and guarded fixed Host. Assets are supplied locally; assembly itself makes no network request and executes no target file.

The actual successful run plus one necessary recovery is in ../windows-runtime-preparation/assembly.log and layout.log. The recovery resolved the source's sherpa ^1.12.28 range to the single already pinned1.13.8 asset; it did not change the source dependency range. Script syntax and clean patch application were checked. No test suite, Windows execution, installer, runner, model, account or tunnel was used.

## Remaining boundary

This is a prepared dependency-graph candidate, not an accepted portable deployment. Its relative directory links need preserving or Windows junction restoration by a Windows packaging step; a generic ZIP extractor is not assumed to do that. Windows process/native-load and OS DLL availability must still pass the existing production gates. No .runtime-staged success marker or Windows-verified manifest was fabricated. MyGo EXE/terminal sidecars can be composed separately after selecting the final portable layout; this delivery does not rename the graph into a deceptively launch-ready release folder.

No huge runtime asset, dependency tree or 59MB candidate manifest belongs in Git. Commit only the small script patch and delivery documentation.
