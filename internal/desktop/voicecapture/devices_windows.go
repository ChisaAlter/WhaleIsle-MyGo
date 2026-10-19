//go:build windows

package voicecapture

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	waveCount   = winmm.NewProc("waveInGetNumDevs")
	waveCaps    = winmm.NewProc("waveInGetDevCapsW")
	waveMessage = winmm.NewProc("waveInMessage")
)

type inputCaps struct {
	Manufacturer, Product uint16
	Version               uint32
	Name                  [32]uint16
	Formats               uint32
	Channels, Reserved    uint16
}

// DRV_QUERYDEVICEINTERFACE / SIZE from mmddk.h. Unlike a wave device index,
// this identity survives reordering and cannot silently select another input.
func inputInterface(index uintptr) (string, error) {
	var size uint32
	code, _, _ := waveMessage.Call(index, 0x080d, uintptr(unsafe.Pointer(&size)), 0)
	if err := mmError("microphone interface size", code); err != nil {
		return "", err
	}
	if size < 2 || size > 65536 || size%2 != 0 {
		return "", errors.New("microphone did not expose a device interface")
	}
	buffer := make([]uint16, size/2)
	code, _, _ = waveMessage.Call(index, 0x080c, uintptr(unsafe.Pointer(&buffer[0])), uintptr(size))
	runtime.KeepAlive(buffer)
	if err := mmError("microphone interface", code); err != nil {
		return "", err
	}
	id := windows.UTF16ToString(buffer)
	if id == "" {
		return "", errors.New("microphone did not expose a device interface")
	}
	return id, nil
}

// InputDevices enumerates names/identities without opening a microphone or
// changing system permissions. The caller supplies its own system-default row.
func InputDevices() ([]Device, error) {
	for _, proc := range []*windows.LazyProc{waveCount, waveCaps, waveMessage} {
		if err := proc.Find(); err != nil {
			return nil, err
		}
	}
	count, _, _ := waveCount.Call()
	devices := make([]Device, 0, count)
	for i := uintptr(0); i < count; i++ {
		var caps inputCaps
		code, _, _ := waveCaps.Call(i, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps))
		if err := mmError("microphone capabilities", code); err != nil {
			return nil, err
		}
		id, err := inputInterface(i)
		if err != nil {
			return nil, err
		}
		devices = append(devices, Device{ID: id, Label: windows.UTF16ToString(caps.Name[:])})
	}
	return devices, nil
}

func resolveInputDevice(device Device) (uintptr, error) {
	if device.ID == "" {
		return waveMapper, nil
	}
	devices, err := InputDevices()
	if err != nil {
		return 0, err
	}
	for i, candidate := range devices {
		if candidate.ID == device.ID {
			return uintptr(i), nil
		}
	}
	return 0, errors.New("selected microphone is unavailable")
}

type inputPreview struct {
	capture *recording
	level   atomic.Uint64
}

func (p *inputPreview) Level() float64 { return math.Float64frombits(p.level.Load()) }
func (p *inputPreview) Cancel()        { p.capture.Cancel() }
func (p *inputPreview) Err() error {
	p.capture.mu.Lock()
	defer p.capture.mu.Unlock()
	return p.capture.err
}

func StartPreview(ctx context.Context, device Device) (Preview, error) {
	n, err := openNativeInput(ctx, bytesPerSecond/10, device)
	if err != nil {
		return nil, err
	}
	p := &inputPreview{capture: newRecording(ctx)}
	go n.runPreview(p)
	return p, nil
}

func (n *nativeRecording) runPreview(p *inputPreview) {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var err error
loop:
	for {
		select {
		case <-p.capture.ctx.Done():
			break loop
		case <-p.capture.cancel:
			break loop
		case <-ticker.C:
			if atomic.LoadUint32(&n.header.Flags)&whdrDone == 0 {
				continue
			}
			// Read only returned buffers; no concurrent native write can race
			// the meter. Clear samples before returning the buffer to WinMM.
			count := min(int(n.header.BytesRecorded), len(n.buffer)) / blockAlign
			var sum float64
			for i := 0; i < count; i++ {
				x := float64(int16(binary.LittleEndian.Uint16(n.buffer[i*2:]))) / 32768
				sum += x * x
			}
			level := 0.0
			if count > 0 {
				level = math.Sqrt(sum / float64(count))
			}
			p.level.Store(math.Float64bits(level))
			clear(n.buffer)
			err = n.headerCall(waveUnprepare)
			if err != nil {
				break loop
			}
			n.prepared = false
			n.header.BytesRecorded, n.header.Flags = 0, 0
			if err = n.headerCall(wavePrepare); err != nil {
				break loop
			}
			n.prepared = true
			if err = n.headerCall(waveAddBuffer); err != nil {
				break loop
			}
		}
	}
	_, cleanup := n.release(false)
	p.level.Store(0)
	p.capture.finish(nil, errors.Join(err, cleanup))
}
