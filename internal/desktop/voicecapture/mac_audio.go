package voicecapture

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"runtime"
	"structs"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Layouts and constants below follow AudioQueue.h, CoreAudioTypes.h and
// AudioHardware{,Base}.h (macOS LP64, both amd64 and arm64). All structures cross
// the ABI by pointer. OSStatus/UInt32 are 32-bit, Boolean is 8-bit, CFIndex is int.
type macAudioFormat struct {
	_                                                structs.HostLayout
	SampleRate                                       float64
	FormatID, Flags, BytesPerPacket, FramesPerPacket uint32
	BytesPerFrame, Channels, Bits, Reserved          uint32
}

type macAudioBuffer struct {
	_              structs.HostLayout
	Capacity       uint32
	Data           unsafe.Pointer
	Length         uint32
	UserData       unsafe.Pointer
	PacketCapacity uint32
	Packets        unsafe.Pointer
	PacketCount    uint32
}

type macPropertyAddress struct {
	_                        structs.HostLayout
	Selector, Scope, Element uint32
}

const (
	macScopeGlobal uint32 = 'g'<<24 | 'l'<<16 | 'o'<<8 | 'b'
	macScopeInput  uint32 = 'i'<<24 | 'n'<<16 | 'p'<<8 | 't'
	macDevices     uint32 = 'd'<<24 | 'e'<<16 | 'v'<<8 | '#'
	macStreams     uint32 = 's'<<24 | 't'<<16 | 'm'<<8 | '#'
	macDeviceUID   uint32 = 'u'<<24 | 'i'<<16 | 'd'<<8 | ' '
	macDeviceName  uint32 = 'l'<<24 | 'n'<<16 | 'a'<<8 | 'm'
	macQueueDevice uint32 = 'a'<<24 | 'q'<<16 | 'c'<<8 | 'd'
	macPCM         uint32 = 'l'<<24 | 'p'<<16 | 'c'<<8 | 'm'
	macUTF8        uint32 = 0x08000100
	macBufferBytes        = bytesPerSecond / 10
)

// Injectable native calls let behavioral fixtures run without framework loads,
// permission prompts, devices, or secrets. No native call retains Go memory.
type macAudioAPI struct {
	permission    func(context.Context) error
	callback      uintptr
	newInput      func(*macAudioFormat, uintptr, uintptr, unsafe.Pointer, unsafe.Pointer, uint32, *unsafe.Pointer) int32
	allocate      func(unsafe.Pointer, uint32, **macAudioBuffer) int32
	enqueue       func(unsafe.Pointer, *macAudioBuffer, uint32, unsafe.Pointer) int32
	start         func(unsafe.Pointer, unsafe.Pointer) int32
	stop, dispose func(unsafe.Pointer, uint8) int32
	setProperty   func(unsafe.Pointer, uint32, unsafe.Pointer, uint32) int32
	propertySize  func(uint32, *macPropertyAddress, uint32, unsafe.Pointer, *uint32) int32
	propertyData  func(uint32, *macPropertyAddress, uint32, unsafe.Pointer, *uint32, unsafe.Pointer) int32
	stringCreate  func(unsafe.Pointer, *byte, int, uint32, bool) unsafe.Pointer
	stringLength  func(unsafe.Pointer) int
	stringMaxSize func(int, uint32) int
	stringCString func(unsafe.Pointer, *byte, int, uint32) uint8
	getType       func(unsafe.Pointer) uintptr
	stringType    func() uintptr
	release       func(unsafe.Pointer)
}

func macStatus(operation string, status int32) error {
	if status == 0 {
		return nil
	}
	return fmt.Errorf("macOS microphone %s failed (OSStatus %d)", operation, status)
}

func (a *macAudioAPI) cfString(s string) unsafe.Pointer {
	b := []byte(s)
	p := a.stringCreate(nil, unsafe.SliceData(b), len(b), macUTF8, false)
	runtime.KeepAlive(b)
	return p
}

func (a *macAudioAPI) goString(p unsafe.Pointer) (string, error) {
	if p == nil || a.getType(p) != a.stringType() {
		return "", errors.New("macOS microphone returned a non-string property")
	}
	length := a.stringLength(p)
	if length < 0 || length > 16384 {
		return "", errors.New("macOS microphone string property is too large")
	}
	size := a.stringMaxSize(length, macUTF8)
	if size < 0 || size > 65536 {
		return "", errors.New("macOS microphone UTF-8 property is too large")
	}
	b := make([]byte, size+1)
	if a.stringCString(p, &b[0], len(b), macUTF8) == 0 {
		return "", errors.New("macOS microphone cannot decode a UTF-8 property")
	}
	for i, v := range b {
		if v == 0 {
			return string(b[:i]), nil
		}
	}
	return "", errors.New("macOS microphone property has no terminator")
}

func (a *macAudioAPI) ids(object, selector, scope uint32) ([]uint32, error) {
	address := macPropertyAddress{Selector: selector, Scope: scope}
	// Hotplug can change the list between size and read. Retry with a fresh
	// bounded allocation rather than truncating identities or indexing stale IDs.
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		var size uint32
		if err := macStatus("property size", a.propertySize(object, &address, 0, nil, &size)); err != nil {
			return nil, err
		}
		if size > 65536 || size%4 != 0 {
			return nil, errors.New("macOS microphone returned an invalid object-list size")
		}
		if size == 0 {
			return nil, nil
		}
		ids := make([]uint32, size/4)
		used := size
		last = macStatus("property data", a.propertyData(object, &address, 0, nil, &used, unsafe.Pointer(&ids[0])))
		if last != nil {
			continue
		}
		if used > size || used%4 != 0 {
			return nil, errors.New("macOS microphone returned an invalid object-list length")
		}
		return ids[:used/4], nil
	}
	return nil, last
}

func (a *macAudioAPI) deviceString(object, selector uint32) (string, error) {
	address := macPropertyAddress{Selector: selector, Scope: macScopeGlobal}
	var p unsafe.Pointer
	size := uint32(unsafe.Sizeof(p))
	status := a.propertyData(object, &address, 0, nil, &size, unsafe.Pointer(&p))
	// HAL explicitly transfers ownership of the name/UID CFString to the caller.
	if p != nil {
		defer a.release(p)
	}
	if err := macStatus("device string", status); err != nil {
		return "", err
	}
	if size != uint32(unsafe.Sizeof(p)) {
		return "", errors.New("macOS microphone returned an invalid string-reference size")
	}
	return a.goString(p)
}

func (a *macAudioAPI) inputDevices() ([]Device, error) {
	objects, err := a.ids(1, macDevices, macScopeGlobal) // kAudioObjectSystemObject
	if err != nil {
		return nil, err
	}
	devices := make([]Device, 0, len(objects))
	seen := make(map[string]bool)
	for _, object := range objects {
		streams, err := a.ids(object, macStreams, macScopeInput)
		if err != nil {
			return nil, err
		}
		if len(streams) == 0 {
			continue
		}
		uid, err := a.deviceString(object, macDeviceUID)
		if err != nil {
			return nil, err
		}
		if uid == "" || seen[uid] {
			return nil, errors.New("macOS microphone returned an empty or duplicate device UID")
		}
		name, err := a.deviceString(object, macDeviceName)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = uid
		}
		seen[uid] = true
		devices = append(devices, Device{ID: uid, Label: name})
	}
	return devices, nil
}

// Native userdata is a monotonically increasing token, never a Go pointer. The
// registry owns callback state until synchronous AudioQueueDispose succeeds.
var macInputs sync.Map
var macInputSequence atomic.Uint64

func macInputCallback(token uintptr, queue unsafe.Pointer, buffer *macAudioBuffer) {
	if value, ok := macInputs.Load(token); ok {
		value.(*macNativeInput).input(queue, buffer)
	}
}

type macAudioController struct {
	load        func() (*macAudioAPI, error)
	mu          sync.Mutex
	busy        bool
	quarantined *macNativeInput
}

type macNativeInput struct {
	api               *macAudioAPI
	owner             *macAudioController
	queue             unsafe.Pointer
	token             uintptr
	mu                sync.Mutex
	pcm               []byte
	length            int
	preview           bool
	level             atomic.Uint64
	stopping, discard bool
	err               error
	finished          chan struct{}
	finishOnce        sync.Once
}

func (n *macNativeInput) signal() { n.finishOnce.Do(func() { close(n.finished) }) }

func (n *macNativeInput) input(queue unsafe.Pointer, b *macAudioBuffer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if b == nil || b.Length > b.Capacity || b.Capacity > macBufferBytes || (b.Length > 0 && b.Data == nil) {
		n.err = errors.New("macOS microphone returned an invalid audio buffer")
		n.signal()
		return
	}
	data := unsafe.Slice((*byte)(b.Data), int(b.Length))
	if n.discard || n.err != nil {
		clear(data)
		return
	}
	if n.preview {
		var sum float64
		count := len(data) / blockAlign
		for i := 0; i < count; i++ {
			x := float64(int16(binary.LittleEndian.Uint16(data[i*blockAlign:]))) / 32768
			sum += x * x
		}
		level := 0.0
		if count > 0 {
			level = math.Sqrt(sum / float64(count))
		}
		n.level.Store(math.Float64bits(level))
	} else {
		count := min(len(data), len(n.pcm)-n.length)
		count -= count % blockAlign
		copy(n.pcm[n.length:], data[:count])
		n.length += count
		if n.length == len(n.pcm) {
			n.signal()
		}
	}
	// Wipe before returning ownership to AudioQueue. Clearing after Enqueue
	// would race the next native write, even on a serialized callback thread.
	clear(data)
	if !n.stopping && (n.preview || n.length < len(n.pcm)) {
		if err := macStatus("enqueue", n.api.enqueue(queue, b, 0, nil)); err != nil {
			n.err = err
			n.signal()
		}
	}
}

func (c *macAudioController) open(ctx context.Context, capacity int, device Device, preview bool) (_ *macNativeInput, err error) {
	if ctx == nil {
		return nil, errors.New("recording context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.quarantined != nil {
		c.mu.Unlock()
		return nil, errors.New("macOS microphone native state is quarantined; restart the application")
	}
	if c.busy {
		c.mu.Unlock()
		return nil, errors.New("microphone recording or preview is already active")
	}
	c.busy = true
	c.mu.Unlock()
	owned := true
	defer func() {
		if owned {
			c.mu.Lock()
			c.busy = false
			c.mu.Unlock()
		}
	}()
	a, err := c.load()
	if err != nil {
		return nil, err
	}
	if device.ID != "" {
		devices, err := a.inputDevices()
		if err != nil {
			return nil, err
		}
		found := false
		for _, candidate := range devices {
			if candidate.ID == device.ID {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("selected microphone is unavailable")
		}
	}
	if err := a.permission(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := &macNativeInput{api: a, owner: c, preview: preview, finished: make(chan struct{})}
	if !preview {
		n.pcm = make([]byte, capacity)
	}
	n.token = uintptr(macInputSequence.Add(1))
	macInputs.Store(n.token, n)
	format := macAudioFormat{SampleRate: sampleRate, FormatID: macPCM, Flags: (1 << 2) | (1 << 3),
		BytesPerPacket: blockAlign, FramesPerPacket: 1, BytesPerFrame: blockAlign, Channels: 1, Bits: 16}
	// NULL runloop uses AudioQueue's internal threads. purego attaches incoming
	// callback threads to Go. The format/out pointer are used synchronously only.
	err = macStatus("new input", a.newInput(&format, a.callback, n.token, nil, nil, 0, &n.queue))
	if err == nil && n.queue == nil {
		err = errors.New("macOS microphone returned a null input queue")
	}
	if err == nil && device.ID != "" {
		uid := a.cfString(device.ID)
		if uid == nil {
			err = errors.New("macOS microphone cannot allocate device UID")
		} else {
			err = macStatus("select device", a.setProperty(n.queue, macQueueDevice, unsafe.Pointer(&uid), uint32(unsafe.Sizeof(uid))))
			a.release(uid)
		}
	}
	for i := 0; i < 3 && err == nil; i++ {
		var buffer *macAudioBuffer
		err = macStatus("allocate buffer", a.allocate(n.queue, macBufferBytes, &buffer))
		if err == nil {
			if buffer == nil {
				err = errors.New("macOS microphone returned a null buffer")
			} else {
				err = macStatus("enqueue buffer", a.enqueue(n.queue, buffer, 0, nil))
			}
		}
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = macStatus("start", a.start(n.queue, nil))
	}
	owned = false // release owns busy and the registry from this point on.
	if err != nil {
		_, cleanup := n.release(false)
		return nil, errors.Join(err, cleanup)
	}
	return n, nil
}

// release never holds the callback lock across Stop/Dispose: synchronous Stop
// can deliver partial buffers. Successful disposal guarantees no later callback.
func (n *macNativeInput) release(keep bool) ([]byte, error) {
	n.mu.Lock()
	n.stopping = true
	n.discard = !keep
	if !keep {
		clear(n.pcm)
		n.length = 0
	}
	n.mu.Unlock()
	var stopErr, disposeErr error
	if n.queue != nil {
		stopErr = macStatus("stop", n.api.stop(n.queue, 1))
		disposeErr = macStatus("dispose", n.api.dispose(n.queue, 1))
	}
	n.mu.Lock()
	var pcm []byte
	err := errors.Join(n.err, stopErr, disposeErr)
	if err == nil && keep && !n.preview {
		pcm = append([]byte(nil), n.pcm[:n.length]...)
	}
	clear(n.pcm)
	n.length = 0
	n.discard = true
	n.level.Store(0)
	if disposeErr == nil {
		n.pcm = nil
		n.queue = nil
		macInputs.Delete(n.token)
	}
	n.mu.Unlock()
	n.owner.mu.Lock()
	if disposeErr != nil {
		// Retain this one bounded callback allocation and fail closed. Native
		// buffers may still be owned by a faulty driver; never free them early.
		n.owner.quarantined = n
		err = fmt.Errorf("%w; native microphone state quarantined", err)
	}
	n.owner.busy = false
	n.owner.mu.Unlock()
	return pcm, err
}

func (n *macNativeInput) run(c *recording, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-c.ctx.Done():
	case <-c.cancel:
	case <-c.stop:
	case <-timer.C:
	case <-n.finished:
	}
	pcm, err := n.release(c.ctx.Err() == nil)
	c.finish(pcm, err)
}

type macInputPreview struct {
	native  *macNativeInput
	capture *recording
}

func (p *macInputPreview) Level() float64 { return math.Float64frombits(p.native.level.Load()) }
func (p *macInputPreview) Cancel()        { p.capture.Cancel() }
func (p *macInputPreview) Err() error {
	p.native.mu.Lock()
	defer p.native.mu.Unlock()
	return errors.Join(p.native.err, p.captureError())
}
func (p *macInputPreview) captureError() error {
	p.capture.mu.Lock()
	defer p.capture.mu.Unlock()
	return p.capture.err
}

func (n *macNativeInput) runPreview(c *recording) {
	select {
	case <-c.ctx.Done():
	case <-c.cancel:
	case <-n.finished:
	}
	_, err := n.release(false)
	c.finish(nil, err)
}
