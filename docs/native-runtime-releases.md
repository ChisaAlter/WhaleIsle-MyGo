# Native first installation and release selection

The launcher now has an actual native release catalog, tag selection, verified
download progress, cancellation, retained failure state and retained staged
installers. Installing a selected tag does not resolve "latest" again. The final
Install and quit action uses the existing session-save / Host task admission /
last-moment installer hash check. Downloading alone neither replaces nor starts
the runtime.

## Runtime versus desktop

MyGo is the native desktop executable. Host and its matching Node are installed
under runtime/vendor and runtime/node by the existing MyGo NSIS package. A
missing-runtime installation selects only a full native package with matching
Host source and readiness-guard metadata. It cannot use an Electron installer,
a WhaleBridge component, a source archive, or the unverified portable candidate.
Bundled Node is resolved before a global PATH Node after installation. An explicit
WHALEISLE_VENDOR override remains the user's responsibility and is not overwritten.

NSIS now puts runtime contents under the runtime directory (and resources under
resources), rather than accidentally using the install-root output path.

## Publisher contract

Publish WhaleIsle-MyGo-releases.json with the native release assets. The catalog
uses schemaVersion=1, application=whaleisle-mygo and a releases array. Each entry
contains the existing updater version/url/sha256/size/notes fields plus tag,
platform=windows, arch=amd64 or arm64, format=nsis, and runtimeIncluded.
Full packages also require hostSourceCommit and hostReadinessGuard matching this
build. Unsupported rows remain visible with a reason; installation is disabled.

After staging, building and signing the native installer, run:

    node scripts/native-release-catalog.mjs release-config.json

Example config (replace paths/version with the actual built release):

    {
      "stage": "dist/whaleisle",
      "installer": "dist/WhaleIsle-Setup-1.2.3.exe",
      "version": "1.2.3",
      "tag": "v1.2.3",
      "arch": "amd64",
      "baseURL": "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.3/",
      "output": "dist/WhaleIsle-MyGo-releases.json"
    }

Optional previous points to the prior native catalog to retain older tags.
Generate after signing because signing changes the installer hash. The publisher
only reads/hashes local artifacts and existing Host/mobile manifests. It never
activates an unverified candidate or runs target packages. A runtime entry is
emitted only for an existing production .runtime-staged package that passes
current Host/mobile verification and includes a matching PE Node binary.

The default GitHub catalog is this repository's latest release asset.
WHALEISLE_RELEASE_CATALOG overrides it; route-specific overrides are
WHALEISLE_RELEASE_CATALOG_GITHUB and WHALEISLE_RELEASE_CATALOG_CNB.
Existing WHALEISLE_UPDATE_FEED variants may point to the same native catalog.
Legacy untyped feeds are not accepted by these launcher install paths.
CNB requires an explicitly configured native catalog and never substitutes the
original Electron repository. All metadata/assets and redirects require HTTPS.

## Delivery boundary

At implementation time this repository has no published releases. A compatible
native installer/catalog must be produced and published before the UI can
complete a real first installation. The existing Windows portable candidate is
still unverified and is not advertised as installable. The feature reports that
absence instead of claiming installation success.
