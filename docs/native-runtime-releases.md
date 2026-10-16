# Native full packages, runtime variants, mirrors and range deltas

The launcher selects an exact native package, downloads or reconstructs it,
checks the final SHA-256/size and configured signature requirements, and retains
the installer until the user confirms the existing task/session-safe handoff.
Downloading does not change the installation.

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

## Execution and publication boundary

This implementation does not establish that an installer, mirror, or delta has
been built, signed, uploaded or published. Runtime use needs those real artifacts
and configured URLs. No publisher execution, update download, installation,
test, build or independent review was performed during this functional delivery.
