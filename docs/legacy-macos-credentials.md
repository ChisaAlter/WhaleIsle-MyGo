# PR01: original macOS credential envelopes

`credentials.Load` reads the original `safeStorage-v1` JSON envelope on macOS via a read-only lookup of service `Whale Isle Safe Storage`, account `Whale Isle`. The bridge uses `SecItemCopyMatching`; it does not add, update, enumerate, or change access controls. macOS may require the user to approve access. Denial or absence returns an error without overwriting the file. No real credentials were accessed during implementation.

Supported payload: original synchronous Electron `v10`, AES-128-CBC, PBKDF2-HMAC-SHA1 with 1003 iterations and the upstream fixed salt/IV, strict PKCS#7 validation followed by the existing strict credentials JSON decoder. CBC is the original unauthenticated legacy format, not a new storage recommendation. Loading does not migrate/rewrite storage. An explicit subsequent save uses the existing native Keychain AES-256-GCM writer.

Unknown prefixes, unavailable Keychain, cgo-disabled builds and differently named legacy application Keychain items remain unsupported. No Chrome/Chromium/Electron fallback item is queried and no key is created. Windows retains its existing DPAPI + GCM path.

## Source contract

- Original `src/main/config.js:361–448`: synchronous safeStorage envelope/read/write.
- Original `src/shared/product-identity.js:6`: product identity `Whale Isle`.
- [Electron 44.3.0 safe-storage implementation](https://github.com/electron/electron/blob/v44.3.0/shell/browser/api/electron_api_safe_storage.cc): synchronous decrypt calls OSCrypt.
- [Electron 44.3.0 browser initialization](https://github.com/electron/electron/blob/v44.3.0/shell/browser/electron_browser_main_parts.cc): service and account naming.
- [Electron 44.3.0 restored synchronous Chromium backend](https://github.com/electron/electron/blob/v44.3.0/patches/chromium/revert_oscrypt_remove_sync_backend.patch): macOS payload format and key derivation. Electron restores this backend after its removal in the Chromium version declared by its DEPS file.

## Verification scope

Synthetic known-answer fixture generated independently with Python hashlib and OpenSSL; no original application execution. Native macOS Security framework execution/compilation requires a macOS SDK and remains a desktop acceptance gate.
