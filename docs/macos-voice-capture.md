# macOS microphone capture

Source implementation: darwin+cgo only. The former !windows stub now remains
only for Linux/other platforms and macOS builds without cgo. No fake audio or
success fallback is used. Native AudioQueue records default-input mono PCM16 at
16 kHz into a preallocated bounded C buffer. Go retains the existing bounded
WAV/Stop/Cancel contract, time limit, context cancellation and consume-once rule.
Queue callbacks never contain Go pointers. Synchronous disposal precedes copying
and zeroing native samples. Unrecoverable disposal failures quarantine callback
state, wipe retained samples, and disable further capture until application exit.

Start checks the app usage description and asks AVCaptureDevice for ordinary OS
microphone consent only after the user's Record action. Canceling a pending
consent prompt does not open an input device after the prompt completes. A small
refcount retains the permission callback state until both caller and callback
release it. The OS consent dialog itself belongs to macOS and cannot be dismissed
programmatically by this controller. Denied/restricted/missing-description states
return errors. No preferences, TCC database or permissions are changed by tests.

Packaging:
- Build the desktop on macOS with the installed Apple SDK and CGO_ENABLED=1
  (`CGO_ENABLED=1 go build -mod=readonly -o whaleisle ./cmd/whaleisle`).
- The cgo source links AVFoundation, AudioToolbox and Foundation automatically.
- mygo.json includes NSMicrophoneUsageDescription and the audio-input entitlement
  template. package-macos.mjs also emits the usage description for its staged app.
- The existing external hardened-runtime signing workflow must merge
  native/macos-audio.entitlements.plist into the main executable's entitlements.
  Do not give microphone permission to the HTML preview helper. This change does
  not sign, install, notarize or create system permissions.

Verification in this Linux workspace: the bounded synthetic PCM lifecycle/WAV
fixture passed; it never calls Start. Packaging JS syntax check passed. There is
no Xcode/xcrun/macOS SDK locally, so Objective-C compilation and actual recording
are NOT claimed. The separate voice-native-build workflow is restricted to this
development branch and voice module paths. It uses the same pinned actions as
preview-native-build and performs only darwin+cgo `go test -c` for amd64/arm64.
It does not execute binaries or build the full application. Native CI compile
result is pending publication; permission denial, real devices, stop/cancel and
resource release still need macOS runtime acceptance.

Public API references:
- https://developer.apple.com/documentation/audiotoolbox/audioqueuenewinput(_:_:_:_:_:_:_:)
- https://developer.apple.com/documentation/avfoundation/avcapturedevice/requestaccess(for:completionhandler:)
