//go:build darwin && cgo

package voicecapture

/*
#cgo CFLAGS: -fblocks
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "capture_darwin.h"
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var macMicrophone sync.Mutex
var macMicrophoneQuarantined bool

// Start is reached only from the explicit Record action. It uses the system
// default input and normal macOS consent; no TCC database or settings are edited.
func Start(ctx context.Context, limits Limits) (Capture, error) {
	if ctx == nil {
		return nil, errors.New("recording context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	capacity, duration, err := boundedLimits(limits)
	if err != nil {
		return nil, err
	}
	if !macMicrophone.TryLock() {
		return nil, errors.New("microphone recording is already active")
	}
	if macMicrophoneQuarantined {
		macMicrophone.Unlock()
		return nil, errors.New("microphone native state is quarantined; restart the application")
	}
	owned := true
	defer func() {
		if owned {
			macMicrophone.Unlock()
		}
	}()
	permission := C.whale_mic_permission()
	if permission == nil {
		return nil, errors.New("cannot allocate microphone permission request")
	}
	defer C.whale_mic_permission_release(permission)
	poll := time.NewTicker(25 * time.Millisecond)
	defer poll.Stop()
	for C.whale_mic_permission_status(permission) == 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-poll.C:
		}
	}
	switch C.whale_mic_permission_status(permission) {
	case -2:
		return nil, errors.New("application bundle lacks NSMicrophoneUsageDescription")
	case -1:
		return nil, errors.New("microphone access denied; review macOS Privacy & Security settings")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var nativeError C.int
	native := C.whale_mic_start(C.size_t(capacity), &nativeError)
	if native == nil {
		if nativeError == -2 {
			macMicrophoneQuarantined = true
		}
		return nil, fmt.Errorf("macOS microphone start failed (AudioQueue %d)", int(nativeError))
	}
	c := newRecording(ctx)
	owned = false
	go func() {
		defer macMicrophone.Unlock()
		timer := time.NewTimer(duration)
		defer timer.Stop()
		tick := time.NewTicker(25 * time.Millisecond)
		defer tick.Stop()
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case <-c.cancel:
				break loop
			case <-c.stop:
				break loop
			case <-timer.C:
				break loop
			case <-tick.C:
				if C.whale_mic_finished(native) != 0 {
					break loop
				}
			}
		}
		if code := C.whale_mic_close(native); code != 0 {
			macMicrophoneQuarantined = true
			c.finish(nil, fmt.Errorf("macOS microphone release failed (AudioQueue %d); native state quarantined", int(code)))
			return
		}
		var count C.size_t
		data := C.whale_mic_bytes(native, &count)
		pcm := C.GoBytes(data, C.int(count))
		C.whale_mic_free(native)
		c.finish(pcm, nil)
	}()
	return c, nil
}
