# macOS isolated preview: optional bundle packaging

Status: packaging integration only. Isolated HTML rendering on macOS is still
unavailable. This change does not create a renderer, alter readiness, change
HelperPath, or permit HTML to be sent to a macOS child process.

## Existing packaging entry point

The `stage` operation in `scripts/package-macos.mjs` accepts an optional
`previewHelperBundle` configuration value pointing to an explicitly supplied,
already-signed `.app`. Omit it to preserve existing package behavior. The
packager does not build or sign a helper, access a keychain, install credentials,
launch an application, or notarize anything.

A supplied helper must have:

- CFBundleIdentifier `org.whaleisle.preview`
- CFBundleExecutable `whaleisle-preview`
- CFBundlePackageType `APPL`
- A Mach-O executable matching the selected package architecture
- A valid Apple-issued code signature, with a concrete TeamIdentifier
- Exactly one entitlement: `com.apple.security.app-sandbox` set to boolean true
- Regular files/directories only; no symlinks or special files

The entitlement allowlist excludes network client/server, user-selected files,
App Sandbox inheritance, app groups, temporary exceptions and debug access.
An independent sandboxed helper must not inherit the unsandboxed host's rights.
The app is copied with `ditto` into
`Contents/Helpers/WhaleIsle Preview.app` and verified again at the destination.
Its bundle identifier and signing team are recorded in the package manifest.

The external signing workflow must preserve this helper's restrictive
entitlements. The `archive` operation rechecks the embedded helper and the
host's signature, and requires both to belong to the same signing team before
creating the archive. Do not use a blanket deep-sign operation that replaces
nested helper entitlements with the host's. Packages without a helper do not
acquire any new signing requirement.

## Remaining rendering boundary

Packaging validation is not runtime sandbox attestation. There is currently no
macOS helper implementation suitable for this contract. MyGo's native
BrowserProcessID implementation is Windows-only. Its macOS IsolatedPreview
window uses a nonpersistent WKWebsiteDataStore; that is storage behavior, not a
process isolation proof. WKWebView's public interface does not provide the
WebContent/Networking service identity and containment checks required by the
current pre-document protocol. A process-parent walk alone is insufficient for
XPC services. This increment adds no private selectors or private sandbox SPI.

Before enabling macOS document delivery, implement and validate the actual
signed helper runtime, trustworthy launch identity, helper and WebKit service
containment before document delivery, bounded byte-only IPC, lifecycle cleanup,
and rendering/capture integration. If those checks cannot be established on a
target system, preserve the static preview and fail closed. Neither a valid
signature nor a private data store may substitute for these checks.

## Verification performed

A focused Node test covers the entitlement allowlist and signing-team parser;
`node --check scripts/package-macos.mjs` checks packaging entry syntax. Real
macOS codesign/plutil/ditto execution, signing, archives and WebKit rendering
were not exercised in the Linux development environment.
