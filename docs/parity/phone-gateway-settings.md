# AX201: native phone gateway settings

The native Remote panel now has an explicit **Edit phone gateway** draft for
mode (relay/LAN/existing SSH mode), listen port, IPv4 bind address, relay endpoint
and relay TLS. Saving validates the entire draft and uses the existing atomic
config store. Invalid input remains in the draft with an error; an editor based
on changed gateway settings must reload. Cancel does not mutate configuration.
Credentials, paired-device identities, remote enablement and unrelated settings
are not written by this editor. The existing default-relay normalization still
forces TLS for the built-in relay. All-interface binding and trusted-relay
warnings are shown beside the controls.

After a successful save, the desktop immediately revokes the existing phone Git
grant, then serializes stop/start through the existing phone runtime lifecycle.
The restarted runtime reads the latest persisted settings; disabled remote stays
disabled. Existing runtime status/error messages remain the readiness signal.
Saving configuration alone does not claim that a daemon or relay connected.
Stopping before refresh avoids losing a configuration edit during startup.

## Verification

On 2026-10-17 one focused Go invocation ran
`go test ./internal/config ./internal/ui/remote ./cmd/whaleisle -run '^TestRemoteGatewayPreferences' -count=1`.
The configuration test passed; the UI and command packages compiled with no
matching tests. This verifies validation, stale-editor rejection and preservation
of unrelated settings at the source-test level. It does not verify Windows/macOS
GUI, native Android, real relay, rebinding or device pairing.

The native Android PR payload mismatch (AX202) is separate and unchanged.
No original WhaleIsle or Android source is modified by this feature.
