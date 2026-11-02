# macOS microphone capture

macOS `Start`, `StartDevice`, `InputDevices`, and `StartPreview` now use native
AudioToolbox/CoreAudio/AVFoundation APIs through purego. Both `CGO_ENABLED=0` and
cgo builds use the same Go controller and microphone lock. The previous
Objective-C bridge remains available as source reference, but is not called by
the exported controller. Linux/other unsupported platforms still report an error.
There is no shell recorder, fake audio, disk recording, or success fallback.

## Ownership and audio contract

AudioQueue converts the selected input to interleaved, little-endian signed
PCM16, mono, 16 kHz. The existing complete-WAV byte limit, duration cap, Stop,
Cancel, context cancellation, and consume-once contract remain in force. A
preallocated Go PCM buffer is bounded by those limits. Three native 3,200-byte
AudioQueue buffers are owned by AudioQueue, never backed by Go memory. Callback
userdata is a monotonically increasing integer token into a Go registry, rather
than a pointer to Go state. The single process-lifetime callback trampoline does
not consume a new purego callback slot per recording.

Callbacks serialize access to the PCM buffer, clear returned native samples
before re-enqueuing, and propagate enqueue/buffer failures to the owner. Stop
marks the controller as stopping before calling synchronous AudioQueueStop, so
partial buffers can be collected without re-enqueuing. Neither Stop nor Dispose
runs while holding the callback mutex. Synchronous AudioQueueDispose then
precedes copying the result and removing callback state. On failure to dispose,
the bounded Go state remains registered, Go PCM is cleared, further returned
samples are discarded/cleared, and all future recording/preview attempts fail
until application restart. A faulty driver can still own native queue buffers;
these are never freed or accessed prematurely.

The preview uses the same exclusive input controller. It computes RMS in [0,1]
from each returned buffer, clears it before returning ownership, and retains only
the level. It has no WAV/export/transcription operation and no accumulated PCM.
Cancel/context cancellation releases the queue and resets the meter.

## Device identity and permission

InputDevices queries HAL object IDs, filters to devices with input streams, and
copies the device UID/name CFStrings with their documented release ownership.
Enumeration does not request consent or open a recording queue. The saved device
ID is the persistent HAL UID, not the transient numeric AudioObjectID. StartDevice
revalidates it against the current roster and sets
`kAudioQueueProperty_CurrentDevice` before enqueuing or starting. A disconnected
saved choice returns an error; it never silently records a different device.
Empty ID follows the system default input.

Only the user's Record or explicit microphone-test action reaches authorization.
NSBundle must contain a nonempty NSString NSMicrophoneUsageDescription. Restricted
and denied states return an actionable error. An undetermined state uses
AVCaptureDevice's normal requestAccessForMediaType completion block. Concurrent
waiters share at most one pending permission request. Cancellation stops waiting;
macOS owns the dialog and its copied block until completion. A late result only
publishes permission and cannot open an input for a canceled context. A fresh
explicit action is required to record.

Each Foundation autorelease pool is created and drained on one locked OS thread.
These Foundation/AVCaptureDevice class methods do not require the application UI
thread; their completion runs on an arbitrary dispatch queue. The permission block
uses purego's copy/dispose implementation. It encodes BOOL as signed char on
amd64 and bool on arm64, matching Apple's Objective-C ABI. CoreFoundation
CFIndex is signed pointer-sized, OSStatus/UInt32 are 32-bit, Boolean is 8-bit;
all audio structures cross the ABI by pointer. A fixture checks LP64 size/offsets,
and the cgo build adds Apple SDK compile-time assertions for ASBD, AudioQueueBuffer,
AudioObjectPropertyAddress, and primitive widths.

## Packaging

- The normal no-cgo build is supported: `CGO_ENABLED=0 go build -mod=readonly -o whaleisle ./cmd/whaleisle`.
- purego lazily opens only absolute paths in `/System/Library/Frameworks`; no SDK,
  shell helper, or runtime build is needed for that build. cgo builds retain the
  framework linkage needed by the reference bridge and SDK ABI assertions.
- The app bundle must contain NSMicrophoneUsageDescription. Existing mygo.json
  and package-macos.mjs supply it. Hardened-runtime signing must include
  `com.apple.security.device.audio-input` from the audio entitlement template.
- Permission belongs to the main desktop executable. Do not assign microphone
  access to an HTML preview helper. This module does not sign, install, notarize,
  change preferences, or edit TCC permissions.

## Verification and remaining acceptance

Development host: Windows, no macOS machine or Apple SDK. All new tests inject
function tables; none calls the real framework loader, touches a microphone, or
changes permissions. The Windows credentials/voicecapture test suites and vet
passed. Both packages also passed no-cgo darwin amd64/arm64 `go test -c` and
`go vet`; these compile/link checks do not execute a macOS binary. After the
parallel composer/chat/pet work settled, the complete `./cmd/whaleisle` application
also passed no-cgo `go build` and `go vet` for both darwin amd64 and arm64. Hidden
runner logs are `tmp/mac-native-check-final.log` (fixtures/packages) and
`tmp/mac-native-app-check-both.log` (whole application). Outputs are independent
`tmp/task6-whaleisle-darwin-{amd64,arm64}` binaries. No macOS runtime acceptance or
cgo/SDK compilation is claimed.

The fixtures cover byte-exact WAV output and clipping, partial-buffer Stop,
Cancel/context cancellation, consume-once, preview RMS/no retained PCM, exclusive
capture/preview ownership, persistent UID selection across device reordering,
absent-device failure, CFString release, malformed buffer rejection, enqueue/start
failure, disposal quarantine, and denied/restricted/missing-description/canceled
permission states. A pending authorization can outlive canceled waiters without
opening a device or allocating another block.

Lead/macOS acceptance must still run the following on both amd64 and arm64:

1. Build the full application with `CGO_ENABLED=0`. Separately build with cgo and
   an installed Apple SDK to exercise the native static assertions and retained
   Objective-C source. Run the fixture suites without real API invocation.
2. Use a signed test app bundle with its actual usage description/entitlements.
   Record with fresh consent, already granted, denied, restricted, and missing
   description states. Confirm missing-description handling returns an error
   before any request, rather than allowing macOS to terminate the process.
3. Cancel while the first OS consent dialog is pending, then grant it. Confirm no
   input opens and no recording indicator appears until a new Record action.
   Retry while the same prompt is pending to check that only one request exists.
4. Enumerate built-in, USB/Bluetooth and any aggregate devices, including Unicode
   names. Select a nondefault input, reorder/reconnect devices, disconnect the
   saved device, and change system default. Confirm the UID is honored and missing
   devices fail. Check the required 16 kHz format on actual device/driver pairs.
5. Record audible speech; inspect mono PCM16/16 kHz WAV and timing/size caps. Test
   short partial-buffer Stop, repeated Stop, Cancel, context cancellation,
   overlapping Record/Preview rejection, and rapid start/stop for leaks/crashes.
6. Test preview start/cancel and record-after-preview. Confirm the meter resets,
   no audio artifact is created, and Activity Monitor's recording indicator
   disappears after successful cleanup. Use Instruments/leaks/ASan as available
   to inspect queue/block/CF object lifetime under repeated cycles. Forced native
   errors are fixture-covered; do not induce them by corrupting a real driver.

## API/ABI references

- [Apple AudioQueueNewInput](https://developer.apple.com/documentation/audiotoolbox/audioqueuenewinput(_:_:_:_:_:_:_:))
- [Apple AudioQueue SDK header](https://github.com/phracker/MacOSX-SDKs/blob/master/MacOSX11.3.sdk/System/Library/Frameworks/AudioToolbox.framework/Versions/A/Headers/AudioQueue.h): callback thread selection, AudioQueueBuffer layout, synchronous Stop/Dispose guarantees, CurrentDevice UID property.
- [Apple CoreAudio HAL headers](https://github.com/phracker/MacOSX-SDKs/blob/master/MacOSX11.3.sdk/System/Library/Frameworks/CoreAudio.framework/Versions/A/Headers/AudioHardwareBase.h): UID/name ownership, input stream/property constants.
- [Apple AVCaptureDevice header](https://github.com/phracker/MacOSX-SDKs/blob/master/MacOSX11.3.sdk/System/Library/Frameworks/AVFoundation.framework/Versions/A/Headers/AVCaptureDevice.h): authorization statuses, escaping completion block and arbitrary dispatch queue.
- [Apple objc.h](https://github.com/phracker/MacOSX-SDKs/blob/master/MacOSX11.3.sdk/usr/include/objc/objc.h) and [Clang AArch64 target](https://github.com/llvm/llvm-project/blob/main/clang/lib/Basic/Targets/AArch64.cpp): BOOL encoding by target.
- [purego v0.11.1 Objective-C block implementation](https://github.com/ebitengine/purego/blob/v0.11.1/objc/objc_block_darwin.go): copy/dispose ownership and shared callback signature cache.
