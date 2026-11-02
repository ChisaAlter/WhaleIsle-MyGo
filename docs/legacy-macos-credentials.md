# Original macOS credential envelopes and native Keychain storage

`credentials.Load` reads the original `safeStorage-v1` JSON envelope on macOS via
a read-only lookup of service `Whale Isle Safe Storage`, account `Whale Isle`.
Both cgo and no-cgo builds support it. The no-cgo bridge binds Security.framework
`SecItemCopyMatching` and CoreFoundation functions with purego; it does not use
`security` commands, shell input/output, plaintext files, or substitute stores.
macOS may ask the user to approve access. Denial or absence returns an error
without overwriting the file. No real credentials or Keychain were accessed
during this task.

Supported legacy payload: original synchronous Electron `v10`, AES-128-CBC,
PBKDF2-HMAC-SHA1 with 1003 iterations and the upstream fixed salt/IV, strict
PKCS#7 validation followed by the existing strict credentials JSON decoder. CBC
is the original unauthenticated legacy format, not a new storage recommendation.
Loading does not migrate/rewrite storage. An explicit subsequent save uses the
native Keychain AES-256-GCM writer.

The writer retains its service `WhaleIsle-MyGo credentials AES-256-GCM`, account
`profile-<SHA256(clean absolute profile directory)>`, and distinct
`macOS-Keychain-AES256GCM-v1` envelope. The complete 32-byte AES key resides in
Keychain, while the profile file holds only AES-GCM ciphertext. A new key is added
only when saving a profile that does not already have this native envelope.
Duplicate creation races re-read the winning Keychain item. Missing/locked keys
for existing envelopes do not cause replacement or plaintext fallback. Denied
writes leave prior file bytes intact. No ACL, synchronizability, or shared access
group changes are made by this bridge.

Unknown prefixes, unavailable/inaccessible Keychain, and differently named
legacy application items remain unsupported. No Chrome/Chromium/Electron fallback
item is queried, no unrelated accounts are enumerated, and the legacy reader
never adds/updates/deletes a key. Windows retains its existing DPAPI + GCM path.
Linux secure storage is still unsupported.

## Native ABI and ownership

Framework loads are lazy and use only absolute Apple system framework paths.
Metadata inspection/import does not load or query a Keychain item. Successful
framework handles remain loaded for the function table's process lifetime.
OSStatus is explicitly int32, CFIndex signed pointer-sized, and CFTypeID unsigned
pointer-sized on the supported LP64 architectures. Exported Security CFStringRef
variables and kCFBooleanTrue are dereferenced once; exported dictionary callback
structures are passed by address. All Security/CoreFoundation calls are
synchronous and copy/retain their inputs before returning; none retains Go memory.

CF dictionary callbacks retain service/account/data objects. All created or copied
CF objects are balanced with CFRelease on success and errors. Reads require CFData,
exactly 32 bytes for the native AES key, or 1..4096 bytes for the legacy secret,
before copying into Go. The CFData owner lives through the copy and is then
released; immutable Security-owned bytes are never modified. Temporary Go key,
legacy-secret, and serialized plaintext buffers are cleared when no longer needed,
including failed and duplicate creation paths.

## Source contract

- Original `src/main/config.js:361–448`: synchronous safeStorage envelope/read/write.
- Original `src/shared/product-identity.js:6`: product identity `Whale Isle`.
- [Electron 44.3.0 safe-storage implementation](https://github.com/electron/electron/blob/v44.3.0/shell/browser/api/electron_api_safe_storage.cc): synchronous decrypt calls OSCrypt.
- [Electron 44.3.0 browser initialization](https://github.com/electron/electron/blob/v44.3.0/shell/browser/electron_browser_main_parts.cc): service and account naming.
- [Electron 44.3.0 restored synchronous Chromium backend](https://github.com/electron/electron/blob/v44.3.0/patches/chromium/revert_oscrypt_remove_sync_backend.patch): macOS payload format and key derivation.
- [Apple SecItem header](https://github.com/apple-oss-distributions/Security/blob/main/keychain/headers/SecItem.h): query keys, result type/ownership, SecItemCopyMatching and SecItemAdd declarations.
- [purego function ABI and memory contract](https://github.com/ebitengine/purego/blob/v0.11.1/func.go): RegisterFunc widths and pointer/string lifetimes.

## Verification scope and macOS follow-up

Development host: Windows without a macOS machine/Apple SDK. Windows fixtures and
vet pass. No-cgo test packages compile for darwin amd64 and arm64; these are cross
compilations, not macOS executions. After concurrent composer/chat/pet work
settled, the full `./cmd/whaleisle` application passed no-cgo `go build` and `go vet`
for both darwin architectures. Evidence: `tmp/mac-native-check-final.log` and
`tmp/mac-native-app-check-both.log`; independent app outputs are
`tmp/task6-whaleisle-darwin-{amd64,arm64}`. Native Security execution and cgo/SDK
compilation remain unverified here.

Behavior fixtures inject a CoreFoundation/Security function table and verify exact
service/account identities, one-item data queries, reference retention/release,
result copying before release, denial/not-found/duplicate status handling,
malformed type/length rejection, read-only legacy access, pre-query envelope
validation, secret zeroing, AES-GCM round trip, and denied/missing-key preservation
of existing profile files. The synthetic CBC known-answer vector was independently
generated with Python hashlib and OpenSSL. No fixture uses user credentials.

Lead/macOS acceptance requirements, on both amd64 and arm64:

1. Build the complete desktop with `CGO_ENABLED=0`, then with cgo and the installed
   Apple SDK. Run only the injected tests (`TestNative*`, `TestKeychain*`, and the
   synthetic legacy tests) without touching a real item. The older DPAPI fixture
   tests are Windows-specific and are not a Keychain runtime acceptance test.
2. In an explicitly authorized disposable macOS test account/profile, save a
   synthetic key/token, restart the app, and load it. Verify one exact service and
   profile-hash account and AES-GCM-only profile contents. Run the same profile
   between cgo/no-cgo builds to verify interoperability.
3. With a disposable test item, exercise locked Keychain, denied/canceled access,
   missing item for an existing native envelope, and concurrent first saves.
   Verify errors and unchanged ciphertext on failures, no replacement key for an
   existing native envelope, and correct duplicate-race recovery.
4. If legacy compatibility must be exercised, provision only a synthetic legacy
   test item under the documented Whale Isle service/account in that isolated
   account. Verify v10 decode, refusal of wrong/unknown/truncated payloads, no
   mutation during Load, and a distinct native envelope after explicit Save.
5. Use Instruments/leaks or a debug allocator to inspect repeated allowed/denied
   read/write operations and CF ownership. Confirm Keychain prompts and signed
   application's ACL behavior under actual packaging/signing identity. Do not
   use the developer's existing Keychain secrets for this verification.
