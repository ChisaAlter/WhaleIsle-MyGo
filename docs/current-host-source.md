# Optional current Host assembly

The default `scripts/build-harness.ps1` keeps the customized Electron source selected
by `WHALEISLE_ELECTRON_SOURCE` (or `reference/electron`). Nothing in this module
replaces that checkout or silently switches an existing vendor runtime.

To build the fixed official source separately:

```powershell
.\scripts\build-harness.ps1 -SourceMode current -CurrentSourceRoot C:\sources\WhaleIsle-90465b67 -BuildOnly
```

The source must be a clean tracked checkout at
`90465b67ec16765631d54c195af19d9ff3c3a770` with the entire Harness build closure.
Use a fresh work directory for each attempt. `build-current-host.ps1` accepts
`-WorkRoot` and `-VendorTarget`; it installs pnpm 11.8.0 locally, honors the frozen
lockfile, and runs the unmodified upstream `build:official` profile. It never
starts Host, creates credentials, or changes the user's profile. The original
Project package and desktop composition source are copied verbatim. Shared
package version strings are not accepted as compatibility evidence.

Without `-BuildOnly`, staging requires an existing customized runtime/vendor.
The new runtime is written separately to runtime/vendor-current-90465b67;
customized plugins and the existing mobile runtime are retained. This does not
claim that the legacy mobile runtime has been upgraded. The build manifest stores
the source commit, source path, platform/architecture and critical artifact hashes.
Source availability alone never writes a successful build manifest.

Select the staged runtime explicitly using `WHALEISLE_VENDOR`. `package.ps1`
uses this selected vendor directory and verifies its current Host manifest before
packaging; its existing mobile runtime check remains required. The older default
runtime remains intact. Do not mix output from different platforms/architectures.

`harness.ReadCurrentHostCapabilities` fails closed for absent/mismatched manifests,
missing/changed artifacts, or a platform mismatch. Only after successful staging
can `ArtifactsReady` and `VisualReplyReader` become true. Project and native Team
also need the actual Project bridge and successfully prepared original desktop
Project overlay. HTML generation additionally requires a real HTML preview bridge
and verified visual-reply isolation. Reader/source/save support is separate from
HTML execution; it cannot qualify generation. All capability defaults remain off.

The Project bridge module calls the original `ensureDesktopDshProject` adapter,
using the selected vendor's dsh-project source, deepseek-harness peer instances,
and profile directory. The adapter owns its overlay and never replaces the user
patch. This module does not reimplement Team, Project RPCs, or HTML authorization.

Dependency denials must be reported directly. Do not replace upstream tarball URLs,
rewrite the lockfile, or use unofficial mirrors to make the build appear complete.

## Verified build result (2026-10-14)

The pinned official Linux x64 build completed native-system, Host, Client and Web with exit 0. The manifest verified 12 critical artifact SHA256 values and the upstream build recorded 386 client artifacts. No Host, daemon, account, GUI or test suite was run. This Linux build cannot satisfy the Windows platform gate; a matching Windows build and actual bridge/isolation readiness remain required.

The completed Linux artifacts also passed temporary staging and staged-manifest verification. The unchanged official Project overlay helper returned success; all seven Project peers resolved within the staged Harness. These checks did not start Host and do not supply a Windows runtime.
