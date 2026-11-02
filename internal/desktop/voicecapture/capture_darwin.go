//go:build darwin

package voicecapture

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var macMicrophone = macAudioController{load: loadMacAudioAPI}

// Start and StartDevice are reached only by an explicit recording action.
// CoreAudio and Keychain are never replaced by shell commands or file helpers.
func Start(ctx context.Context, limits Limits) (Capture, error) {
	return StartDevice(ctx, limits, Device{})
}

func StartDevice(ctx context.Context, limits Limits, device Device) (Capture, error) {
	capacity, duration, err := boundedLimits(limits)
	if err != nil {
		return nil, err
	}
	n, err := macMicrophone.open(ctx, capacity, device, false)
	if err != nil {
		return nil, err
	}
	c := newRecording(ctx)
	go n.run(c, duration)
	return c, nil
}

// InputDevices reads HAL identities and names without requesting microphone
// consent or opening an input. Empty ID is the caller-provided system-default row.
func InputDevices() ([]Device, error) {
	a, err := loadMacAudioAPI()
	if err != nil {
		return nil, err
	}
	return a.inputDevices()
}

func StartPreview(ctx context.Context, device Device) (Preview, error) {
	n, err := macMicrophone.open(ctx, 0, device, true)
	if err != nil {
		return nil, err
	}
	c := newRecording(ctx)
	p := &macInputPreview{native: n, capture: c}
	go n.runPreview(c)
	return p, nil
}

// Handles and callback trampolines are process-lifetime resources. Framework
// initialization is lazy, and only Apple's absolute system paths are loaded.
var loadMacAudioAPI = sync.OnceValues(func() (_ *macAudioAPI, err error) {
	handles := make(map[string]uintptr)
	for _, framework := range []string{"CoreFoundation", "CoreAudio", "AudioToolbox", "Foundation", "AVFoundation"} {
		path := "/System/Library/Frameworks/" + framework + ".framework/" + framework
		handle, e := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
		if e != nil {
			for _, h := range handles {
				_ = purego.Dlclose(h)
			}
			return nil, fmt.Errorf("macOS microphone load %s: %w", framework, e)
		}
		handles[framework] = handle
	}
	defer func() {
		if err != nil {
			for _, h := range handles {
				_ = purego.Dlclose(h)
			}
		}
	}()
	a := &macAudioAPI{}
	for _, binding := range []struct {
		framework, name string
		to              any
	}{
		{"AudioToolbox", "AudioQueueNewInput", &a.newInput},
		{"AudioToolbox", "AudioQueueAllocateBuffer", &a.allocate},
		{"AudioToolbox", "AudioQueueEnqueueBuffer", &a.enqueue},
		{"AudioToolbox", "AudioQueueStart", &a.start},
		{"AudioToolbox", "AudioQueueStop", &a.stop},
		{"AudioToolbox", "AudioQueueDispose", &a.dispose},
		{"AudioToolbox", "AudioQueueSetProperty", &a.setProperty},
		{"CoreAudio", "AudioObjectGetPropertyDataSize", &a.propertySize},
		{"CoreAudio", "AudioObjectGetPropertyData", &a.propertyData},
		{"CoreFoundation", "CFStringCreateWithBytes", &a.stringCreate},
		{"CoreFoundation", "CFStringGetLength", &a.stringLength},
		{"CoreFoundation", "CFStringGetMaximumSizeForEncoding", &a.stringMaxSize},
		{"CoreFoundation", "CFStringGetCString", &a.stringCString},
		{"CoreFoundation", "CFGetTypeID", &a.getType},
		{"CoreFoundation", "CFStringGetTypeID", &a.stringType},
		{"CoreFoundation", "CFRelease", &a.release},
	} {
		address, e := purego.Dlsym(handles[binding.framework], binding.name)
		if e != nil {
			return nil, fmt.Errorf("macOS microphone resolve %s: %w", binding.name, e)
		}
		purego.RegisterFunc(binding.to, address)
	}
	mediaAddress, err := purego.Dlsym(handles["AVFoundation"], "AVMediaTypeAudio")
	if err != nil {
		return nil, fmt.Errorf("macOS microphone resolve AVMediaTypeAudio: %w", err)
	}
	mediaPointer := *(*unsafe.Pointer)(unsafe.Pointer(&mediaAddress))
	media := *(*objc.ID)(mediaPointer) // exported NSString *const, not the address of the variable
	device := objc.ID(objc.GetClass("AVCaptureDevice"))
	bundleClass := objc.ID(objc.GetClass("NSBundle"))
	stringClass := objc.GetClass("NSString")
	poolClass := objc.ID(objc.GetClass("NSAutoreleasePool"))
	if media == 0 || device == 0 || bundleClass == 0 || stringClass == 0 || poolClass == 0 {
		return nil, errors.New("macOS microphone authorization classes are unavailable")
	}
	newSelector := objc.RegisterName("new")
	drain := objc.RegisterName("drain")
	mainBundle := objc.RegisterName("mainBundle")
	infoKey := objc.RegisterName("objectForInfoDictionaryKey:")
	isKind := objc.RegisterName("isKindOfClass:")
	length := objc.RegisterName("length")
	status := objc.RegisterName("authorizationStatusForMediaType:")
	request := objc.RegisterName("requestAccessForMediaType:completionHandler:")
	permission := &macPermission{
		usage: func() error {
			// An autorelease pool must be created/drained on the same OS thread.
			// These Foundation/AVCaptureDevice class APIs have no main-thread
			// affinity; the completion handler may use any dispatch queue.
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			pool := poolClass.Send(newSelector)
			defer pool.Send(drain)
			key := a.cfString("NSMicrophoneUsageDescription")
			if key == nil {
				return errors.New("cannot allocate microphone usage-description key")
			}
			defer a.release(key)
			bundle := bundleClass.Send(mainBundle)
			reason := bundle.Send(infoKey, key)
			if reason == 0 || !objc.Send[bool](reason, isKind, stringClass) || objc.Send[uintptr](reason, length) == 0 {
				return errors.New("application bundle lacks NSMicrophoneUsageDescription")
			}
			return nil
		},
		status: func() int64 {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			pool := poolClass.Send(newSelector)
			defer pool.Send(drain)
			return objc.Send[int64](device, status, media) // NSInteger is signed 64-bit on LP64.
		},
		request: func(done func(bool)) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			pool := poolClass.Send(newSelector)
			defer pool.Send(drain)
			// purego manages the block's copy/dispose ABI and Go closure cache.
			// AVFoundation copies this escaping block. Release our own reference
			// after the message, even if ctx is canceled while its dialog is open.
			// BOOL is signed char on amd64 and bool on arm64. Match both the
			// invocation ABI and the block's Objective-C type encoding.
			block := newMacPermissionBlock(done)
			defer block.Release()
			device.Send(request, media, block)
		},
	}
	a.permission = permission.authorize
	a.callback = purego.NewCallback(func(token uintptr, queue unsafe.Pointer, buffer *macAudioBuffer, _ unsafe.Pointer, _ uint32, _ unsafe.Pointer) {
		macInputCallback(token, queue, buffer)
	})
	return a, nil
})
