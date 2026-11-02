//go:build darwin && cgo

package voicecapture

/*
#cgo CFLAGS: -fblocks
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include <AudioToolbox/AudioToolbox.h>
#include <CoreAudio/CoreAudio.h>
#include <stddef.h>

// When an Apple SDK is available, reject ABI drift at compile time. The Go
// layouts are separately asserted by a fixture on both supported architectures.
_Static_assert(sizeof(AudioStreamBasicDescription) == 40, "ASBD size");
_Static_assert(offsetof(AudioStreamBasicDescription, mFormatID) == 8, "ASBD format offset");
_Static_assert(offsetof(AudioStreamBasicDescription, mReserved) == 36, "ASBD reserved offset");
_Static_assert(sizeof(AudioQueueBuffer) == 56, "AudioQueueBuffer size");
_Static_assert(offsetof(AudioQueueBuffer, mAudioData) == 8, "AQ buffer data offset");
_Static_assert(offsetof(AudioQueueBuffer, mAudioDataByteSize) == 16, "AQ buffer length offset");
_Static_assert(offsetof(AudioQueueBuffer, mUserData) == 24, "AQ buffer userdata offset");
_Static_assert(offsetof(AudioQueueBuffer, mPacketDescriptions) == 40, "AQ buffer packets offset");
_Static_assert(offsetof(AudioQueueBuffer, mPacketDescriptionCount) == 48, "AQ buffer packet count offset");
_Static_assert(sizeof(AudioObjectPropertyAddress) == 12, "HAL property address size");
_Static_assert(sizeof(OSStatus) == 4, "OSStatus width");
_Static_assert(sizeof(Boolean) == 1, "Boolean width");
_Static_assert(sizeof(CFIndex) == 8, "CFIndex width");
*/
import "C"
