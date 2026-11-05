# Native full packages, runtime variants, mirrors and range deltas

The launcher selects an exact native package, downloads or reconstructs it,
checks the final SHA-256/size and configured signature requirements, and retains
the installer until the user confirms the existing task/session-safe handoff.
Downloading does not change the installation.

For the current parity work, use the [LIFE-025 publication handoff](#life-025-publication-handoff-2026-11-03).
The dated local artifacts below are historical evidence, not the current parity
candidate or proof that either public update source is deployed.

## Package identity and runtime

Catalog schemaVersion=2, application=whaleisle-mygo, releases=[] adds:

- packageKind: full or delta
- packageID: tag/platform/arch/full/runtime or tag/platform/arch/full/desktop
- Delta IDs replace full with delta and append baseVersion/baseSHA256
- downloadURLs: explicitly configured github/cnb URLs for the same bytes

The existing version/tag/sha256/size/notes, platform, arch, format,
runtimeIncluded, hostSourceCommit and hostReadinessGuard fields remain.
Windows uses windows + nsis; macOS uses darwin + dmg. Architectures are amd64
and arm64. Size and SHA-256 always describe the complete final installer,
including when packageKind=delta.

A tag may contain both desktop-only and runtime-inclusive installers. A missing
Host permits only a runtime-inclusive package with the matching Host source and
readiness guard. A complete runtime package includes the desktop, Host and Node;
it is not a standalone Host archive. An explicit WHALEISLE_VENDOR remains under
the user's control. The installer does not replace that external directory.

Windows also accepts the installed rc.2 profile with the exact upstream commit
`687f282e8625c6b8d11ff58211a90524be2e4317` paired with
`mygo-session-runtime-readiness-v1`. The publisher verifies the actual Session
reader exports, strict unknown-event rejection, pinned readiness output, Node
hash, Project artifacts and the complete release graph. The old source-build
commits still require `mygo-visual-readiness-v1`; the guards cannot be mixed.
A present but invalid source-build manifest never falls back to installed proof.

Create an independent release stage with:

    node scripts/session-runtime.mjs release runtime tmp/fresh-release-runtime

This copies the selected vendor graph without following dependency aliases,
relocates internal junctions, and records installed mobile resources only in the
new copy. The mobile probe loads SQLite against an in-memory database and loads
the actual PTY bindings; it never creates a terminal, daemon, device identity,
listening socket or profile. Standard Node and Electron-as-Node have separate
recorded versions, ABI, executable hash and loaded native artifact hashes.
Electron resources are required whenever the actual probe reports Electron.
The mobile manifest also pins the exact rc.2 Host peer used by its ACP fallback.
All physical files and links must match the manifest; an extra code file fails.
The release publisher and installer restoration verify the complete vendor graph.
Host launch keeps the existing pinned core Session/Node/readiness preflight rather
than hashing all dependency files on every start; the phone resource resolver
verifies its separate mobile inventory before selecting the recorded executable.

`package.ps1 -OutputRoot tmp/fresh-package` keeps existing dist artifacts. It
refuses an existing stage. For rc.2 it uses this release stage, compiles the
windowsgui dependency-link helper, and emits an exact NSIS physical-file include.
NSIS does not recursively expand pnpm links: the installer restores only declared
internal aliases after promotion, and detaches verified owned aliases before
rename, rollback or uninstall. Unknown/changed reparse points fail before any
link mutation. Existing F/D payload ownership guards remain in force.

Schema v1 catalogs are still readable as full packages, with their original
single tag/target identity. The publisher migrates retained v1 entries to full
v2 identities. Legacy Feed.blockMap/blockMapURL alone never enable native deltas.
Unsupported packages remain visible with their reason and cannot be selected.

## Publisher

Use already built and signed native artifacts. This command reads/hashes them
and emits metadata; it does not build, publish, upload, download, install, or
execute an app or installer:

    node scripts/native-release-catalog.mjs release-config.json

Example with both Windows variants at the same tag:

    {
      "version": "1.2.4",
      "tag": "v1.2.4",
      "baseURL": "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.4/",
      "previous": "previous/WhaleIsle-MyGo-releases.json",
      "packages": [
        {
          "stage": "dist/desktop/whaleisle",
          "installer": "dist/WhaleIsle-Desktop-1.2.4-amd64.exe",
          "arch": "amd64",
          "deltaBases": [
            {
              "packageID": "v1.2.3/windows/amd64/full/desktop",
              "installer": "previous/WhaleIsle-Desktop-1.2.3-amd64.exe"
            }
          ]
        },
        {
          "stage": "dist/runtime/whaleisle",
          "installer": "dist/WhaleIsle-Runtime-1.2.4-amd64.exe",
          "arch": "amd64"
        }
      ],
      "output": "dist/WhaleIsle-MyGo-releases.json"
    }

The former single stage/installer config remains supported. Each package may
override arch, platform, stage, installer, baseURL, route, mirrorBaseURLs, notes
and deltaBases. Runtime metadata is derived from the actual staged files, not
from a boolean in the config. Windows runtime packages use the existing Host,
mobile and Node checks. macOS uses inspectMacOSApp/inspectDMG from the macOS
packaging module; its app version must match the release. Run platform-specific
verification with the matching platform/architecture as required by the current
Host verifier. Generate metadata after signing because signing changes bytes.

Both variants need distinct installer filenames and distinct stage directories.
Adding/replacing one variant keeps the other variant and other architectures.
Replacing a full variant removes obsolete delta recipes targeting its previous
bytes. The existing 500-entry and 2 MiB catalog limits remain.

## Actual range-delta contract

A delta is an independent package selection using algorithm=sha256-range-v1.
It reuses the existing block-map/HTTP Range reconstruction implementation; it
is not a separate executable or a fabricated patch asset. Its URL points at the
same signed final installer as its full counterpart.

The publisher accepts deltaBases only when the previous catalog identifies the
exact same platform, architecture and runtime variant. It hashes the supplied
old installer and requires the recorded base SHA-256/size. It creates a complete
4 MiB target block map, compares each block with the same offset in the base,
and emits a delta only when at least one byte range can actually be reused.
An installer with no reusable blocks yields only a full entry.

Each delta includes:

- basePackageID, baseVersion, baseSHA256 and baseSize
- fullPackageID, identifying the target full installer
- downloadSize, the exact total of changed ranges before HTTP overhead
- blockMap.fileSize and a contiguous complete list of offset/size/SHA-256 blocks

The client requires the target full entry to match final version/hash/size and
runtime provenance. It persists verified complete installers and identities
under the user-data updates/native-cache directory, keyed by SHA-256. Successfully
reconstructed installers become full bases for a later version too. The running
version, cached identity and size must match before a delta is selectable; the
entire cached base hash is checked before reconstruction starts. An installed
whaleisle.exe is never mistaken for the old NSIS installer.

With no matching cached base, the delta is unavailable and the real full entry
remains selectable. A corrupt/missing base discovered at staging returns the
full-package alternative. HTTP Range unsupported by the selected mirror falls
back to the exact same complete installer, using the same final hash/size,
signature requirements and progress path. Hash or reconstruction failures do not
silently succeed. Cancellation does not replace the installation. Progress during
reconstruction represents complete output bytes, including locally reused blocks.
Cache-write failure leaves the verified installer usable and explicitly says
future updates can use full packages. Removing the cache also leaves full updates
available. No cache files are treated as an installed version by themselves.

For an automatic latest/tag selection, compatible releases are ordered by stable
version, desktop-only before runtime-inclusive when runtime is already present,
then expected transfer size. A missing Host filters out desktop-only packages.
The version page selects the exact packageID, never merely the shared tag.

Windows NSIS is the supported delta target. macOS currently uses complete DMGs.
Opening a verified DMG does not claim installation: the user drags Whale Isle.app
to Applications after the existing confirmed quit/handoff.

## macOS offline Host component

The component page accepts the actual WhaleIsle-Host-<version>-darwin-<arch>.tar.gz
emitted by package-macos.mjs. Its top level is runtime/ plus
WhaleIsle-Host-bundle.json. Electron deepseek-harness.tar, DMGs and desktop-only
packages are not interchangeable with this archive.

“Install offline bundle” asks for the local file and confirms its source and
destination. It extracts into a fresh component staging directory, preserves
relative pnpm/workspace links, and checks the native package identity, Host
provenance/readiness artifacts, macOS Node architecture and matching Node build
record. It installs under userData/components/whaleisle-host/versions without
executing Node or changing the running Host.

“Select installed version” uses a directory picker in that versions directory.
After confirmation, the existing Host task inspection/drain/stop path stops Host
and the phone connection. The atomic installed.json selection points both vendor
and Node at this complete runtime. Previous versions and profile data remain;
selection itself leaves Host stopped. “Start Host” uses the ordinary launcher
start, recreating the supervisor, Project bridge and plugin install control with
the selected paths. Subsequent app starts reuse that selection.

An explicit WHALEISLE_VENDOR, dshBin or nodeBin must be cleared before selecting
a managed Host version; this action never rewrites those overrides. Runtime
selection does not replace the signed .app or grant macOS permissions. Local
bundle import is not a claim of release publication or signing.

## Same-package download sources

baseURL names the HTTPS directory for the package's canonical route. route is
github by default or explicitly cnb. mirrorBaseURLs may provide another route's
HTTPS release directory. The publisher appends the same encoded installer basename
and writes downloadURLs; a mirror therefore uses the full package's single
SHA-256/size identity. Mirror URLs are publisher configuration, not a claim that
an upload or remote-byte verification has happened.

The client's selected route chooses the corresponding downloadURLs entry for
both full downloads and range requests. A v2 mapped package with no selected
mirror is unavailable instead of downloading from a different source silently.
Legacy v1 entries continue to use their route-specific catalog's explicit URL.

The default catalog is the native repository's latest GitHub asset:
WhaleIsle-MyGo-releases.json. WHALEISLE_RELEASE_CATALOG overrides the catalog;
route-specific overrides are WHALEISLE_RELEASE_CATALOG_GITHUB and
WHALEISLE_RELEASE_CATALOG_CNB. Existing WHALEISLE_UPDATE_FEED variants may also
point to a native catalog. CNB requires an explicitly configured catalog URL;
that URL may host a copy of the common catalog containing the publisher's actual
CNB downloadURLs. No official CNB repository/address is inferred or invented.
All metadata, installer URLs and redirects keep the existing HTTPS restrictions.

## LIFE-025 publication handoff (2026-11-03)

The local catalog producer and the client download-source contract are present.
The remaining delivery is a real published native installer/catalog and the
actual CNB target. Neither target's current availability was queried in this
pass. An unset CNB route is unavailable in the UI; this does not complete the
domestic-update feature.

### Inputs the release owner must supply

| Input | Required value |
| --- | --- |
| Accepted candidate | The newly accepted parity stage, its exact embedded desktop version, and the final installer built from that stage. The old rc.3/trajectory diagnostic and dated packages below are not substitutes. |
| Version/tag | The accepted version and one matching tag: the version itself, `v<version>`, or `mygo-v<version>`. Do not change the bundled Host version/source/readiness record to match a desktop version. |
| GitHub target | The client's existing default is `ChisaAlter/WhaleIsle-MyGo`; publish the catalog as `WhaleIsle-MyGo-releases.json` in the release selected by `/releases/latest/download/`. This is the configured target, not an availability check. |
| CNB target | The release owner's approved native repository and its actual anonymous HTTPS attachment-directory URL for this tag, plus the actual anonymous catalog URL. No native CNB address is recorded in this project. |
| Authority | Separate permission to publish/upload and, if required, create the matching tag. Use the approved GitHub/CNB credential mechanism for that target; never put credentials in metadata or URLs. Signing keys remain in the existing signing process. |

The final installer basename and exact size/SHA-256 become the package identity
on both routes. Build the catalog **after signing** from those final bytes and
the actual stage using `scripts/native-release-catalog.mjs`. The existing
producer verifies the selected native Host/mobile graph and Node target for a
runtime-inclusive package. A shell-only package cannot supply a missing Host.

For the already documented producer config, set:

- `version`, `tag`, `stage` and `installer` from that accepted candidate.
- `baseURL` to `https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/<tag>/`.
- `mirrorBaseURLs.cnb` only to the approved, real CNB attachment directory,
  ending in `/`. The producer appends the same encoded installer basename.
- `output` to a new `WhaleIsle-MyGo-releases.json` path. The producer refuses
  an existing output; retain old metadata rather than overwriting it.
- `previous`/`deltaBases` only when retaining a real previous native catalog
  and its exact installer. Full-package publication does not require a delta.

Run the existing producer only when these real inputs exist:

```text
node scripts/native-release-catalog.mjs release-config.json
```

Upload the final installer and catalog to the approved native GitHub release
and CNB release. Any retained historical entries or delta bases in the common
catalog must keep their recorded assets available. Native range deltas use the
same final installer; there is no separate patch binary to upload.

A prerelease must use an explicit catalog override while it remains a
prerelease: the default GitHub `/releases/latest/download/` is a stable-release
discovery path. Do not promote an unaccepted candidate merely to satisfy that
URL. The actual anonymous CNB catalog URL is consumed through
`WHALEISLE_RELEASE_CATALOG_CNB`; a deployment intended to work without launch
environment setup also needs that approved address wired as the product's
default in a separately built candidate. An operator-only environment override
does not provide an out-of-box CNB source.

### Do not publish native artifacts through the Electron mirror unchanged

The original installed product uses `ayasealter/WhaleIsle` on CNB. That is not
an established MyGo-native target. Its `scripts/sync-cnb.mjs` validates stable
Electron/WhaleBridge releases, `Whale-Isle-Setup-<version>.exe`, Electron update
metadata and `SHA512SUMS.txt`; it rejects prereleases. Copying that workflow or
renaming a native catalog cannot satisfy its contract. Keep the repositories
and package identities distinct. Reuse that release system's credential and
upload process only after its owner approves the native target and adapts its
asset validation to this native catalog.

### Evidence that closes LIFE-025

After publication is authorized, the release owner must fetch the catalog and
selected installer anonymously from **each promised route**, verify the final
installer size/SHA-256 against the catalog, and exercise the native client's
catalog selection and download/staging for the current platform/runtime.
Confirm that route changes use that route's mapped installer and Range URL,
not a silent download from the other source. Installation/upgrade and its
task-safe handoff remain LIFE-026 acceptance.

Until those results and any promised default-CNB wiring are present,
LIFE-025 remains incomplete. No release upload, signing, network check,
workflow activation or installer generation was performed for this handoff.

## Execution and publication boundary

This session verified a real independent rc.2 release stage containing 25,695
physical files and 3,146 internal aliases, performed actual mobile SQLite/PTY
loads with Electron 44.5.0 / Node 24.21.0 / ABI 149, built the Windows desktop and
preview helper, and ran the focused Go/JavaScript fixtures and isolated NSIS
transaction fixtures. The five dependency-link review findings were independently
rechecked against source and recorded test evidence. The previous Windows
installer compiled successfully with the updated dependency helper, but **does
not contain the later Windows console-popup fix**. It is historical output and
must not be delivered as the corrected package. Its byte-for-byte archive is
[WhaleIsle-Setup-0.2.1-rc.2-final.exe](../tmp/release-chain-full-package-20261025/archive-pre-console-fix-20261026/WhaleIsle-Setup-0.2.1-rc.2-final.exe):
293,892,796 bytes, SHA-256
`53ee366210c8281345ede5d53046f6213a1da8879904d1a46090efa74a789319`.
Its archived [blockmap](../tmp/release-chain-full-package-20261025/archive-pre-console-fix-20261026/WhaleIsle-Setup-0.2.1-rc.2-final.exe.blockmap.json)
contains 4,485 blocks; the old sum is preserved in
[SHA256SUMS-final.txt](../tmp/release-chain-full-package-20261025/archive-pre-console-fix-20261026/SHA256SUMS-final.txt).
[final-installer-metadata.json](../tmp/release-chain-full-package-20261025/archive-pre-console-fix-20261026/final-installer-metadata.json)
records those old bytes. The runtime dependency helper
has SHA-256 `3a03300e188775869478d36191397b64bb16eb83d8f7d38e29a03ddd59d724e5`.
[archived source hashes](../tmp/release-chain-full-package-20261025/archive-pre-console-fix-20261026/final-binary-source-hashes.json)
pin that helper and its two source files with the previous desktop and preview
executables. An earlier compression using the old helper was canceled; its
partial output is stale, is excluded from delivery and is never used as catalog
input.

The archived [previous catalog](../tmp/release-chain-full-package-20261025/archive-pre-console-fix-20261026/WhaleIsle-MyGo-releases.json)
contains one schema-2 Windows amd64 NSIS full-runtime prerelease for the old package. Its digest,
size and final installer basename were independently matched to the actual byte
metadata. It records the exact rc.2 source commit and Session readiness guard,
with no invented sample catalog entries. Strict publisher revalidation after
the final startup probe accepted the complete vendor graph without broadening
exclusions or deleting extra files. The actual Go client decoded and validated
this catalog with runtime required, and the Go phone resource resolver accepted
the final stage. These checks passed in
[release-chain-catalog.log](../tmp/release-chain-catalog.log). The complete
compiler and final-byte metadata evidence is in
[release-chain-final-setup.log](../tmp/release-chain-final-setup.log).

The console-popup rebuild requires the desktop executable to be frozen with the
same `-X main.buildVersion=<manifest cliVersion>` linker setting as the package
script and a new stage startup observation to be accepted. The first candidate,
SHA-256
`c6180db6d21e6b49cdc8570e9f80dc98e2a030c37afd3d1aa2645bf419143d72`,
reported `dev`; its startup observation also retained two unattributed show
events. It is not accepted for the corrected full installer. The original logs
remain intact. The shell-only NSIS command check passed in 91.676 seconds using
a separate temporary output, but it skips runtime/THA4 expansion and does not
certify a corrected complete package.

The next frozen candidate has SHA-256
`e2ba60bdf832d497fca1313fccb9191bd1d19edb8f63f942f381a14630773f30`,
size 96,824,832 bytes and the package-equivalent version linker setting. Its
[build receipt](../tmp/acceptance-final/final-app.json) and
[versioned stage receipt](../tmp/release-chain-full-package-20261025/console-fix-versioned-stage-receipt.json)
were checked before replacing only the stage app. The subsequent
[startup observation](../tmp/acceptance-startup/run-46a79040/receipt.json)
confirmed runtime appVersion `0.2.1-rc.2`, the adjacent Node/CLI and visible main
window, with zero owned terminal, unattributed terminal, unresolved window or
hook-error events. It nevertheless failed strict acceptance: `WM_CLOSE` was
delivered, but the app did not exit within 60.184 seconds, leaving the owned Job
and two ports live before forced observer cleanup. That failed receipt remains
preserved. Diagnostic runs, including a temporary restore failure later handled
by the Lead, are separate evidence and were not accepted as official startup
success.

The later [official settled-90 observation](../tmp/acceptance-startup/settled-90/run-25b3dfad/receipt.json)
passed against the same frozen e2ba app: runtime appVersion `0.2.1-rc.2`, adjacent
Node/CLI without overrides, ready in 7.944 seconds, and a visible main window.
It observed 98.060 seconds before close, including 90.116 seconds after ready.
`WM_CLOSE` then produced a natural exit code 0 in 5.726 seconds, with no forced
cleanup, an empty owned Job and both owned ports closed. Owned visible terminals,
unattributed terminals, unresolved windows and hook errors were all zero; the
main window SHOW was observed. These results describe the settled observation;
they do not erase or explain away the earlier close failure. The accepted receipt
SHA-256 is `b9cb31f432cf6e4de009d63f79e88694990bcd2170a2829d4722347a96366978`.
The versioned stage receipt binds that digest and sets the actual observed
version, version verification and startup GREEN gates before compression. The
new complete installer uses a fresh `-consolefix.exe` output; all previous
installer bytes and metadata stay archived.

The corrected complete Windows amd64 installer is
[WhaleIsle-Setup-0.2.1-rc.2-consolefix.exe](../tmp/release-chain-full-package-20261025/WhaleIsle-Setup-0.2.1-rc.2-consolefix.exe):
293,890,576 bytes, SHA-256
`0512b3d5f97d06253253c2fef9e218843746341b4ff71cfcff104333b3a7dceb`.
It contains the accepted e2ba desktop executable and the Windows background
process-tree console fix, reports appVersion `0.2.1-rc.2`, and retains the same
preview, runtime dependency helper and physical runtime include. Its
[final-installer-metadata.json](../tmp/release-chain-full-package-20261025/final-installer-metadata.json)
records `containsConsolePopupFix: true` and separately identifies the archived
old package as not containing that fix. All 4,485 64-KiB
[blockmap entries](../tmp/release-chain-full-package-20261025/WhaleIsle-Setup-0.2.1-rc.2-consolefix.exe.blockmap.json)
were regenerated from the corrected bytes, with a shorter final block. The new
[SHA256SUMS-final.txt](../tmp/release-chain-full-package-20261025/SHA256SUMS-final.txt)
and single-entry schema-2 [catalog](../tmp/release-chain-full-package-20261025/WhaleIsle-MyGo-releases.json)
name only the corrected `-consolefix.exe` package. The old same-named sidecars
were moved intact to the archive before these new files were written.

Strict wholegraph publisher verification, independent actual byte hash/size/name
comparison and the actual Go `DecodeNativeCatalog` plus `Validate(true)` and
mobile `ResolveResources` roundtrips all passed. The publisher/client sequence
took 22.988 seconds; see
[release-chain-console-catalog.log](../tmp/release-chain-console-catalog.log),
[its acceptance receipt](../tmp/release-chain-full-package-20261025/console-fix-catalog-acceptance.json)
and the [complete NSIS log](../tmp/release-chain-console-setup.log).
Independent final-byte/blockmap verification and GUID-isolated install, upgrade
and uninstall acceptance are recorded in
[artifact.log](../tmp/acceptance-final/artifact.log),
[nsis-transaction.log](../tmp/acceptance-final/nsis-transaction.log) and the
[acceptance round report](parity/acceptance-round-20261026.md). The product installer
has not been run under the user's product identity; no signing, upload or
publication was performed.

Host launch verifies the pinned core reader, Node and readiness artifacts;
release publication and installer restoration additionally verify the complete
dependency graph. The phone resolver verifies its separate complete mobile
inventory before choosing the recorded Electron executable.

No signing, notarization, release upload, publication, update download or update
installation is implied by local staging and metadata generation. Configured
HTTPS URLs describe intended asset locations; local catalog generation does not
prove those assets exist on the server. The isolated NSIS test uses its own
application identity and temporary directories, without installing over the
user's product or modifying its profile.
## Local validation artifact (2026-10-26 perfround)

The perfround desktop build (`whaleisle.exe` `a175f15cb18d673c948d50fa7863fd09cab9ea2fe0b28d2f4dc36edace4da6e5`,
94,458,368 bytes) was staged with the unchanged Host runtime: runtime 25,698 files /
928,321,379 bytes (Node 93,580,104; mobile `dshd-remote` 7,191 files / 365,412,421 bytes;
runtime graph 25,695 files / 3,146 links), resources 60 / 3,273,495 bytes and THA4 assets
10 / 38,610,327 bytes. The packaging list is complete: 25,773 File entries / 1,100,041,687
bytes; NSIS reported `Install data` 1,084,986,965 bytes after duplicate-content dedupe,
exactly 2,362,880 bytes below the consolefix payload (the desktop/preview delta).

The emitted file `WhaleIsle-Setup-0.2.1-rc.2-perfround.exe` is **not publishable**: it is
239,147,553 bytes with an unfinalized NSIS block header (`length_of_all_following_data` = 0,
expected 239,009,825), its build log stops at `Compressed data:`, and no blockmap, metadata,
`SHA256SUMS-final.txt` or catalog was produced. An uninterrupted rebuild of the exact same
stage produced 291,494,145 bytes (921.1 s); the delivered file is 52,346,592
bytes shorter, i.e. the compressed tail was never written. Another archived full-package file
(254,876,193 bytes) shares the same defect, while `-final.exe` (293,892,796) and
`-consolefix.exe` (293,890,576) are finalized and their sidecars verify.

A GUID-identity full-payload rebuild of the same stage passed the isolated transaction:
silent install produced `whaleisle.exe` = `a175f15c…`, the installed tree matched the staged
payload exactly (25,773 files / 1,100,041,687 bytes, 0 missing / 0 extra / 0 differing),
the same installer re-ran as an upgrade without a retired tree, and silent uninstall removed
the install root, registration and shortcuts. The user's Electron install (`Whale Isle 0.3.10`)
kept its uninstall registration and had zero files written during the test window.

Before any publication of a rebuilt perfround package:

1. Allow the full LZMA pass (about 15–22 minutes; this rebuild measured 921.1 s,
   the consolefix build 1,345 s) and verify the block header equals `fileSize - 137728` and the
   log prints `Total size:`.
2. Recompute the blockmap/metadata/SHA256SUMS from the final bytes, then emit the schema v2
   catalog with a single entry (`runtimeIncluded`, windows/amd64/nsis/full, prerelease) and the
   Host pin `687f282e8625c6b8d11ff58211a90524be2e4317` / `mygo-session-runtime-readiness-v1`.
3. Re-run the 64 KiB block recompute, Go `DecodeNativeCatalog` + `Validate(true)` and mobile
   `ResolveResources`, then the isolated GUID install/upgrade/uninstall transaction.

Reference sidecars currently describe the archived consolefix package: size 293,890,576 bytes,
SHA-256 `0512b3d5f97d06253253c2fef9e218843746341b4ff71cfcff104333b3a7dceb`, 4,485 blocks with zero
mismatches, single-entry schema v2 catalog, and the same Host pin.

## Local validation artifact (2026-10-26 perfround2)

The perf-round package was rebuilt from the current working tree instead of
reusing the previous stage: `scripts/package.ps1` produced a fresh full stage
(792.8 s) with the current desktop and preview binaries, the unchanged Host
runtime, the physical 25,698-entry include and a regenerated include file
byte-identical to the previous one. A complete NSIS LZMA pass (765.8 s, with a
5,400 s timeout) then emitted the finalized artifact:

- `WhaleIsle-Setup-0.2.1-rc.2-perfround2.exe`, 291,585,042 bytes,
  SHA-256 `4a7dce448bccffb1d3dbada0d3d54313075ecbc5f9dd898701d6d606f0e0a2c6`
- NSIS block header `length_of_all_following_data` = 291,447,314 =
  size − 137,728 (finalized), and the log contains
  `Total size: 291,585,042 / 1,096,292,266 bytes (26.5%)`
- staged desktop `da198fb62e3fd1c865383c86ae13a8885cea839b36858b7b1fd208b77894defb`
  (94,509,056 bytes, version `0.2.1-rc.2` embedded) and preview
  `b75041dd4c4a63efed0dd0c228c19d22ad2faa7f9028a698e354927ee0db2fcc`
- runtime 25,698 files / 928,321,379 bytes and the same 25,698-entry include
  (`ccfe6bed…`); resources 60 / 3,273,495 bytes; THA4 assets 10 / 38,610,327 bytes
- 4,450 64-KiB blockmap entries regenerated from the final bytes, every block
  re-verified; `final-installer-metadata.json`, `SHA256SUMS-final.txt` and a
  single-entry schema-2 catalog name only this installer, with the unchanged
  Host pin `687f282e8625c6b8d11ff58211a90524be2e4317` /
  `mygo-session-runtime-readiness-v1`
- Go `DecodeNativeCatalog` + `Validate(true)` and mobile `ResolveResources`
  passed against the actual bytes; the strict wholegraph publisher took 19.1 s

A GUID-identity rebuild of the same stage (`whaleisle.perfround2.<id>`, not the
product identity) passed the four-step transaction: silent install (163.5 s)
produced the exact staged desktop and preview hashes, the installed tree matched
the stage payload exactly (25,773 files / 1,100,111,319 bytes, 0 missing / 0
extra / 0 differing, identical tree digest), reinstalling the same installer
upgraded in place with no retired tree (298.2 s), and silent uninstall removed
the install root, registration, desktop shortcut and start-menu folder (118.7 s).
The real `ai.deepseek.harness.gui` keys stayed absent and the user's Electron
install (`C:\软件\Whale Isle`, 31,551 files / 1,238,327,017 bytes) kept its
fingerprint and uninstall registration with zero files written during the
session. The isolated test receipt initially stored that non-ASCII directory
through a GBK-misread literal; a follow-up cycle with a code-point-built path
re-ran install/uninstall and recorded a valid before/after fingerprint
(`PASS-electron-isolation-supplement`).

Evidence: `tmp/perfround2-20261026/` (stage build, compile log, metadata,
catalog, artifact checks, startup acceptance `startup/run-11d4d669`, transaction,
isolation supplement, preservation inventory) and
`tmp/release-chain-full-package-20261025/` (installer, sidecars, stage receipt).
The earlier `-perfround.exe` (239,147,553 bytes, unfinalized header) and the
archived 254,876,193-byte file remain in place untouched and are not
deliverables.

## Local validation artifact (2026-10-27 final)

The final package was rebuilt from the current working tree after aligning the
native toast motion: a 160 ms fade-in while the toast rises 8 DIP, a 3000 ms
full-opacity hold, and a 1000 ms fade-out, giving the native 4000 ms default
life (`HOLD_MS + FADE_MS`). The action toast keeps its 8000 ms life, fading over
its final second. The checked-in reference stylesheet uses a 6 px downward
offset; the task specified an 8 px upward rise, which is what this build
implements. Unit tests assert 3000/1000/160/8 ms/DIP and the 8 DIP first-frame
offset, and the full `internal/ui` and green suites passed.

- `WhaleIsle-Setup-0.2.1-rc.2-final.exe`, 291,548,253 bytes,
  SHA-256 `2682f61f6a401fe1462445f9bf3365eb234375e9c587227ab7bb2725a4760d69`
- NSIS block header `length_of_all_following_data` = 291,410,525 =
  size − 137,728 (finalized); the log contains
  `Total size: 291,548,253 / 1,096,295,850 bytes (26.5%)` after a 630.2 s
  LZMA pass (4,200 s timeout); `Install data` is 1,085,060,181 bytes
- staged desktop `d2bf513a468ed31430541b3086e983cc2b923d0e863ffe0a000917302fc44ee3`
  (94,512,640 bytes, version `0.2.1-rc.2` embedded) and preview
  `ef8dc2df32fec676d367fe4f1ae25fd036acbb2f5d806c6b943aa31e704b114f`
- the stage came from the full `scripts/package.ps1` chain (689.6 s): runtime
  25,698 files / 928,321,379 bytes identical to the previous stage, 3,146
  reparse points rewritten to the canonical path, graph 25,695 files / 3,146
  links, resources 60 / 3,273,495 bytes, THA4 10 / 38,610,327 bytes, and the
  byte-identical 25,698-entry `runtime-files.nsh`
  (`ccfe6bed…`, 6,899,252 bytes)
- 4,449 64-KiB blockmap entries regenerated from the final bytes, every block
  re-verified (last block 44,125 bytes); `final-installer-metadata.json`,
  `SHA256SUMS-final.txt` and a single-entry schema-2 catalog name only this
  installer, with the unchanged Host pin
  `687f282e8625c6b8d11ff58211a90524be2e4317` /
  `mygo-session-runtime-readiness-v1`
- Go `DecodeNativeCatalog` + `Validate(true)` and mobile `ResolveResources`
  passed against the actual bytes; the strict wholegraph publisher took 18.8 s;
  the independent 16-check artifact audit passed

A GUID-identity rebuild of the same stage (`whaleisle.final.eabbf9011ff7`, not
the product identity; 291,523,416 bytes / `dff95599…` / 627.4 s) passed the
four-step transaction: silent install (142.2 s) produced the exact staged
desktop and preview hashes, the installed tree matched the stage payload exactly
(25,773 files / 1,100,114,903 bytes, 0 missing / 0 extra / 0 differing, digest
`e5fe0fd6…`), reinstalling upgraded in place with no retired tree (290.0 s), and
silent uninstall removed the install root, registration, desktop shortcut and
start-menu folder (116.1 s). The real `ai.deepseek.harness.gui` keys stayed
absent and the user's Electron install (`C:\软件\Whale Isle`, 31,551 files) kept
its uninstall registration and full tree fingerprint (names, sizes and mtimes).

The accepted stage bytes were started once from their own directory before
packaging: Ready 6.422 s, exact owned WM_CLOSE after 90.188 s, natural exit in
5.785 s with code 0, zero owned visible terminals, zero unattributed terminal
events, one main-window show and zero observer hook errors. Both root launchers
(`whaleisle.exe` and `DeepSeekHarness Go.exe`) were synchronized to the staged
desktop `d2bf513a…` / 94,512,640 bytes after the transaction; the pre-sync bytes
are backed up under `tmp/final-20261027/root-backup/`.

Evidence: `tmp/final-20261027/` (stage receipt, compile logs, metadata,
catalog acceptance, 16-check artifact audit, startup acceptance
`startup/run-86a471f7`, transaction, root sync, final check) and
`tmp/release-chain-full-package-20261025/` (installer, sidecars, stage receipt).
The prior `-perfround2.exe` (291,585,042 bytes / `4a7dce44…`) and the unfinalized
`-perfround.exe` (239,147,553 bytes) remain in place untouched; the old sidecars
and stage are archived under `archive-pre-final-20261027/` and are not
deliverables.
