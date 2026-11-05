//go:build windows

package voicecapture

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	handle    uintptr
	header    *waveHeader
	buffer    []byte
	pins      runtime.Pinner
	prepared  bool
	extra     *waveChunk
	nextExtra bool
}

type waveChunk struct {
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
	n, err := openNativeInput(ctx, min(pcmBytes, bytesPerSecond/10), device)
	if err != nil {
		return nil, err
	}
	// Keep a second buffer queued while the returned chunk is copied/metered.
	// A single periodically requeued buffer would drop speech between chunks.
	if err := n.queueSecondChunk(); err != nil {
		_, cleanup := n.release(false)
		return nil, errors.Join(err, cleanup)
	}
	c := newRecording(ctx)
	go n.run(c, deadline, pcmBytes)
	return c, nil
}

func (n *nativeRecording) queueSecondChunk() error {
	chunk := &waveChunk{buffer: make([]byte, len(n.buffer)), header: &waveHeader{}}
	n.extra = chunk
	chunk.pins.Pin(&chunk.buffer[0])
	chunk.pins.Pin(chunk.header)
	chunk.header.Data = uintptr(unsafe.Pointer(&chunk.buffer[0]))
	chunk.header.BufferLength = uint32(len(chunk.buffer))
	if err := n.chunkCall(wavePrepare, chunk.header); err != nil {
		return err
	}
	chunk.prepared = true
	return n.chunkCall(waveAddBuffer, chunk.header)
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
	// Returned 100ms chunks supply the live meter and are requeued in order.
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

func (n *nativeRecording) run(c *recording, deadline time.Time, capacity int) {
	timer := time.NewTimer(max(time.Until(deadline), 0))
	ticker := time.NewTicker(10 * time.Millisecond)
	defer timer.Stop()
	defer ticker.Stop()
	pcm := make([]byte, 0, capacity)
	finish := func(runErr error) {
		tail, err := n.release(c.ctx.Err() == nil)
		if len(pcm) < capacity {
			pcm = append(pcm, tail[:min(len(tail), capacity-len(pcm))]...)
		}
		clear(tail)
		c.finish(pcm, errors.Join(runErr, err))
	}
	for {
		select {
		case <-c.ctx.Done():
			finish(nil)
			return
		case <-c.cancel:
			finish(nil)
			return
		case <-c.stop:
			finish(nil)
			return
		case <-timer.C:
			finish(nil)
			return
		case <-ticker.C:
			header, buffer, prepared := n.header, n.buffer, &n.prepared
			if n.nextExtra {
				header, buffer, prepared = n.extra.header, n.extra.buffer, &n.extra.prepared
			}
			if atomic.LoadUint32(&header.Flags)&whdrDone == 0 {
				continue
			}
			if err := n.chunkCall(waveUnprepare, header); err != nil {
				finish(err)
				return
			}
			*prepared = false
			count := int(atomic.LoadUint32(&header.BytesRecorded))
			if count > len(buffer) {
				finish(errors.New("invalid microphone byte count"))
				return
			}
			count -= count % blockAlign
			c.level.Store(math.Float64bits(pcmLevel(buffer[:count])))
			pcm = append(pcm, buffer[:min(count, capacity-len(pcm))]...)
			// Already-consumed bytes must not be included again by release.
			header.BytesRecorded = 0
			clear(buffer)
			n.nextExtra = !n.nextExtra
			if len(pcm) >= capacity {
				finish(nil)
				return
			}
			header.Flags = 0
			if err := n.chunkCall(wavePrepare, header); err != nil {
				finish(err)
				return
			}
			*prepared = true
			if err := n.chunkCall(waveAddBuffer, header); err != nil {
				finish(err)
				return
			}
		}
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
			appendChunk := func(header *waveHeader, buffer []byte) {
				count := atomic.LoadUint32(&header.BytesRecorded)
				if count > uint32(len(buffer)) {
					dataErr = errors.New("Windows microphone returned an invalid audio byte count")
					return
				}
				count -= count % blockAlign
				pcm = append(pcm, buffer[:int(count)]...)
			}
			if n.nextExtra && n.extra != nil {
				appendChunk(n.extra.header, n.extra.buffer)
			}
			appendChunk(n.header, n.buffer)
			if !n.nextExtra && n.extra != nil {
				appendChunk(n.extra.header, n.extra.buffer)
			}
		}
		clear(n.buffer)
		n.pins.Unpin()
		n.buffer = nil
		n.header = nil
		if n.extra != nil {
			clear(n.extra.buffer)
			n.extra.pins.Unpin()
			n.extra = nil
		}
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
		if n.extra != nil && n.extra.prepared {
			extraErr := n.chunkCall(waveUnprepare, n.extra.header)
			if extraErr == nil {
				n.extra.prepared = false
			}
			unprepareErr = errors.Join(unprepareErr, extraErr)
		}
		if !n.prepared && (n.extra == nil || !n.extra.prepared) {
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
	return n.chunkCall(proc, n.header)
}
func (n *nativeRecording) chunkCall(proc *windows.LazyProc, header *waveHeader) error {
	code, _, _ := proc.Call(n.handle, uintptr(unsafe.Pointer(header)), unsafe.Sizeof(*header))
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
