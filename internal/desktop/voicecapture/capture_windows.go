//go:build windows

package voicecapture

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	waveMapper = 0xffffffff
	whdrDone   = 0x00000001
)

var (
	// Restrict loading to Windows' system DLL directory, never the working
	// directory or a downloaded DLL. Merely importing this package opens no device.
	winmm         = windows.NewLazySystemDLL("winmm.dll")
	waveOpen      = winmm.NewProc("waveInOpen")
	wavePrepare   = winmm.NewProc("waveInPrepareHeader")
	waveAddBuffer = winmm.NewProc("waveInAddBuffer")
	waveStart     = winmm.NewProc("waveInStart")
	waveStop      = winmm.NewProc("waveInStop")
	waveReset     = winmm.NewProc("waveInReset")
	waveUnprepare = winmm.NewProc("waveInUnprepareHeader")
	waveClose     = winmm.NewProc("waveInClose")
	nativeMu      sync.Mutex
	nativeBusy    bool
	failedNative  *nativeRecording
)

// waveFormat matches WAVEFORMATEX. cbSize is zero for ordinary PCM; the native
// function reads the documented fields, not Go's trailing struct alignment.
type waveFormat struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	ExtraSize      uint16
}

// Pointer-sized fields preserve WAVEHDR's Windows 32/64-bit ABI. Data and Next
// are native addresses; the Go owner independently retains and pins the header
// and backing audio buffer for the entire asynchronous native operation.
type waveHeader struct {
	Data          uintptr
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          uintptr
	Reserved      uintptr
}

type nativeRecording struct {
	handle   uintptr
	header   *waveHeader
	buffer   []byte
	pins     runtime.Pinner
	prepared bool
}

// Start opens the Windows-selected input device only when called. There are no
// permission-setting changes, fallbacks to files/URLs, or network operations.
func Start(ctx context.Context, limits Limits) (Capture, error) {
	return StartDevice(ctx, limits, Device{})
}

// StartDevice resolves the saved interface identity against the current device
// roster before opening it; a disconnected choice never falls back to another
// microphone. An empty identity follows the Windows system input.
func StartDevice(ctx context.Context, limits Limits, device Device) (Capture, error) {
	pcmBytes, duration, err := boundedLimits(limits)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(duration)
	n, err := openNativeInput(ctx, pcmBytes, device)
	if err != nil {
		return nil, err
	}
	c := newRecording(ctx)
	go n.run(c, deadline)
	return c, nil
}

func openNativeInput(ctx context.Context, pcmBytes int, device Device) (*nativeRecording, error) {
	if ctx == nil {
		return nil, errors.New("recording context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	deviceID, err := resolveInputDevice(device)
	if err != nil {
		return nil, err
	}
	for _, proc := range []*windows.LazyProc{waveOpen, wavePrepare, waveAddBuffer, waveStart, waveStop, waveReset, waveUnprepare, waveClose} {
		if err := proc.Find(); err != nil {
			return nil, fmt.Errorf("Windows microphone recording is unavailable: %w", err)
		}
	}
	nativeMu.Lock()
	if failedNative != nil {
		nativeMu.Unlock()
		return nil, errors.New("a Windows microphone driver failed to release the previous recording; restart the application before recording again")
	}
	if nativeBusy {
		nativeMu.Unlock()
		return nil, errors.New("another microphone recording is already active")
	}
	nativeBusy = true
	nativeMu.Unlock()

	n := &nativeRecording{buffer: make([]byte, pcmBytes), header: &waveHeader{}}
	n.pins.Pin(&n.buffer[0])
	n.pins.Pin(n.header)
	n.header.Data = uintptr(unsafe.Pointer(&n.buffer[0]))
	n.header.BufferLength = uint32(len(n.buffer))
	format := waveFormat{FormatTag: 1, Channels: 1, SamplesPerSec: sampleRate,
		AvgBytesPerSec: bytesPerSecond, BlockAlign: blockAlign, BitsPerSample: 16}
	// CALLBACK_NULL: bounded recording needs no cross-runtime native callback.
	// A single full-capacity buffer avoids requeue gaps and ordering hazards.
	code, _, _ := waveOpen.Call(uintptr(unsafe.Pointer(&n.handle)), deviceID, uintptr(unsafe.Pointer(&format)), 0, 0, 0)
	if code != 0 {
		_, cleanupErr := n.release(false)
		if code == 32 { // WAVERR_BADFORMAT
			return nil, errors.Join(errors.New("Windows microphone does not support 16000 Hz mono PCM16 recording"), cleanupErr)
		}
		return nil, errors.Join(fmt.Errorf("could not open Windows microphone; check the input device and microphone access settings: %w", mmError("waveInOpen", code)), cleanupErr)
	}
	if err = n.headerCall(wavePrepare); err == nil {
		n.prepared = true
		err = n.headerCall(waveAddBuffer)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = n.deviceCall(waveStart)
	}
	if err != nil {
		_, cleanupErr := n.release(false)
		return nil, errors.Join(err, cleanupErr)
	}
	return n, nil
}

func (n *nativeRecording) run(c *recording, deadline time.Time) {
	timer := time.NewTimer(max(time.Until(deadline), 0))
	ticker := time.NewTicker(10 * time.Millisecond)
	defer timer.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
		case <-c.cancel:
		case <-c.stop:
		case <-timer.C:
		case <-ticker.C:
			// Polling WHDR_DONE is a documented WinMM completion mechanism.
			// Native data is read only after reset and unprepare below.
			if atomic.LoadUint32(&n.header.Flags)&whdrDone == 0 {
				continue
			}
		}
		pcm, err := n.release(c.ctx.Err() == nil)
		c.finish(pcm, err)
		return
	}
}

// release is the only path which unpins the buffers or releases nativeBusy.
// reset returns a partially filled buffer as well as stopping acquisition.
func (n *nativeRecording) release(keep bool) ([]byte, error) {
	cleanupErr := n.shutdown()
	var pcm []byte
	var dataErr error
	if cleanupErr == nil {
		if keep {
			count := atomic.LoadUint32(&n.header.BytesRecorded)
			if count > uint32(len(n.buffer)) {
				dataErr = errors.New("Windows microphone returned an invalid audio byte count")
			} else {
				count -= count % blockAlign
				pcm = append([]byte(nil), n.buffer[:int(count)]...)
			}
		}
		clear(n.buffer)
		n.pins.Unpin()
		n.buffer = nil
		n.header = nil
	}
	nativeMu.Lock()
	if cleanupErr != nil {
		// Never unpin or reclaim memory that a failing native driver may still
		// reference. Retain at most this one bounded allocation, fail closed to
		// further recording, and report the cleanup failure rather than expose
		// its audio. This also keeps Pinner alive on the exceptional path.
		failedNative = n
	}
	nativeBusy = false
	nativeMu.Unlock()
	return pcm, errors.Join(cleanupErr, dataErr)
}

func (n *nativeRecording) shutdown() error {
	if n.handle == 0 {
		return nil
	}
	var resetErr, unprepareErr, closeErr error
	for attempt := 0; attempt < 3; attempt++ {
		resetErr = n.deviceCall(waveReset)
		if resetErr != nil {
			// Try the independent stop operation before retrying buffer return.
			_ = n.deviceCall(waveStop)
		}
		unprepareErr = nil
		if n.prepared {
			unprepareErr = n.headerCall(waveUnprepare)
			if unprepareErr == nil {
				n.prepared = false
			}
		}
		if !n.prepared {
			closeErr = n.deviceCall(waveClose)
			if closeErr == nil {
				n.handle = 0
				return nil
			}
		}
		if attempt < 2 {
			time.Sleep(20 * time.Millisecond)
		}
	}
	return fmt.Errorf("Windows microphone cleanup failed: %w", errors.Join(resetErr, unprepareErr, closeErr))
}

func (n *nativeRecording) headerCall(proc *windows.LazyProc) error {
	code, _, _ := proc.Call(n.handle, uintptr(unsafe.Pointer(n.header)), unsafe.Sizeof(*n.header))
	runtime.KeepAlive(n)
	return mmError(proc.Name, code)
}

func (n *nativeRecording) deviceCall(proc *windows.LazyProc) error {
	code, _, _ := proc.Call(n.handle)
	return mmError(proc.Name, code)
}

func mmError(operation string, code uintptr) error {
	if code == 0 {
		return nil
	}
	return fmt.Errorf("%s failed (Windows multimedia error %d)", operation, code)
}
