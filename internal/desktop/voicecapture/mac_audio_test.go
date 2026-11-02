package voicecapture

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// These fixtures exercise the same API table/state machine as darwin. They do
// not load Apple frameworks, create consent UI, or acquire an audio device.
type macAudioFixture struct {
	api                                       *macAudioAPI
	controller                                *macAudioController
	queue                                     *byte
	buffers                                   []*macAudioBuffer
	backing                                   [][]byte
	token                                     uintptr
	enqueues, starts, stops, disposals        int
	startStatus, enqueueStatus, disposeStatus int32
	partialOnStop                             []byte
	selected                                  string
	strings                                   map[unsafe.Pointer][]byte
	ids                                       []uint32
	input                                     map[uint32]bool
	uids, names                               map[uint32]string
}

func newMacAudioFixture(t *testing.T) *macAudioFixture {
	t.Helper()
	f := &macAudioFixture{queue: new(byte), strings: make(map[unsafe.Pointer][]byte), ids: []uint32{10, 20, 30},
		input: map[uint32]bool{10: true, 20: false, 30: true}, uids: map[uint32]string{10: "builtin-uid", 30: "usb-uid"}, names: map[uint32]string{10: "Built-in 麦克风", 30: "USB microphone"}}
	a := &macAudioAPI{}
	f.api = a
	a.permission = func(context.Context) error { return nil }
	a.newInput = func(format *macAudioFormat, callback, token uintptr, runloop, mode unsafe.Pointer, flags uint32, out *unsafe.Pointer) int32 {
		if format.SampleRate != 16000 || format.FormatID != macPCM || format.Flags != 12 || format.BytesPerPacket != 2 || format.FramesPerPacket != 1 || format.BytesPerFrame != 2 || format.Channels != 1 || format.Bits != 16 || format.Reserved != 0 {
			t.Error("incorrect PCM format")
		}
		if callback != a.callback || runloop != nil || mode != nil || flags != 0 {
			t.Error("incorrect AudioQueue invocation")
		}
		f.token = token
		*out = unsafe.Pointer(f.queue)
		return 0
	}
	a.allocate = func(_ unsafe.Pointer, capacity uint32, out **macAudioBuffer) int32 {
		b := make([]byte, capacity)
		buffer := &macAudioBuffer{Capacity: capacity, Data: unsafe.Pointer(&b[0])}
		f.backing = append(f.backing, b)
		f.buffers = append(f.buffers, buffer)
		*out = buffer
		return 0
	}
	a.enqueue = func(_ unsafe.Pointer, b *macAudioBuffer, packets uint32, descriptions unsafe.Pointer) int32 {
		f.enqueues++
		if packets != 0 || descriptions != nil {
			t.Error("PCM enqueue has packet descriptions")
		}
		if !bytes.Equal(unsafe.Slice((*byte)(b.Data), b.Length), make([]byte, b.Length)) {
			t.Error("native buffer not wiped before enqueue")
		}
		return f.enqueueStatus
	}
	a.start = func(unsafe.Pointer, unsafe.Pointer) int32 { f.starts++; return f.startStatus }
	a.stop = func(_ unsafe.Pointer, immediate uint8) int32 {
		f.stops++
		if immediate != 1 {
			t.Error("stop is not synchronous")
		}
		if f.partialOnStop != nil {
			f.deliver(f.partialOnStop)
		}
		return 0
	}
	a.dispose = func(_ unsafe.Pointer, immediate uint8) int32 {
		f.disposals++
		if immediate != 1 {
			t.Error("dispose is not synchronous")
		}
		return f.disposeStatus
	}
	a.propertySize = func(object uint32, p *macPropertyAddress, qualifierSize uint32, qualifier unsafe.Pointer, size *uint32) int32 {
		if qualifierSize != 0 || qualifier != nil || p.Element != 0 {
			t.Error("incorrect HAL qualifier")
		}
		if object == 1 && p.Selector == macDevices && p.Scope == macScopeGlobal {
			*size = uint32(len(f.ids) * 4)
			return 0
		}
		if p.Selector == macStreams && p.Scope == macScopeInput {
			*size = 0
			if f.input[object] {
				*size = 4
			}
			return 0
		}
		t.Errorf("unexpected HAL size property %#v", p)
		return -1
	}
	createString := func(s string) unsafe.Pointer {
		b := append([]byte(s), 0)
		p := unsafe.Pointer(&b[0])
		f.strings[p] = b
		return p
	}
	a.propertyData = func(object uint32, p *macPropertyAddress, _ uint32, _ unsafe.Pointer, size *uint32, out unsafe.Pointer) int32 {
		switch p.Selector {
		case macDevices:
			copy(unsafe.Slice((*uint32)(out), len(f.ids)), f.ids)
		case macStreams:
			*(*uint32)(out) = 101
		case macDeviceUID:
			*(*unsafe.Pointer)(out) = createString(f.uids[object])
		case macDeviceName:
			*(*unsafe.Pointer)(out) = createString(f.names[object])
		default:
			t.Errorf("unexpected HAL data property %#v", p)
			return -1
		}
		return 0
	}
	a.stringCreate = func(_ unsafe.Pointer, p *byte, count int, encoding uint32, external bool) unsafe.Pointer {
		if encoding != macUTF8 || external {
			t.Error("incorrect CFString creation")
		}
		return createString(string(unsafe.Slice(p, count)))
	}
	a.stringLength = func(p unsafe.Pointer) int { return len(f.strings[p]) - 1 }
	a.stringMaxSize = func(length int, _ uint32) int { return length * 3 }
	a.stringCString = func(p unsafe.Pointer, out *byte, size int, _ uint32) uint8 {
		copy(unsafe.Slice(out, size), f.strings[p])
		return 1
	}
	a.getType = func(unsafe.Pointer) uintptr { return 7 }
	a.stringType = func() uintptr { return 7 }
	a.release = func(p unsafe.Pointer) {
		if _, ok := f.strings[p]; !ok {
			t.Error("release of unowned CFString")
		}
		delete(f.strings, p)
	}
	a.setProperty = func(_ unsafe.Pointer, property uint32, data unsafe.Pointer, size uint32) int32 {
		if property != macQueueDevice || size != uint32(unsafe.Sizeof(unsafe.Pointer(nil))) {
			t.Error("incorrect queue UID property")
		}
		p := *(*unsafe.Pointer)(data)
		f.selected = string(f.strings[p][:len(f.strings[p])-1])
		return 0
	}
	f.controller = &macAudioController{load: func() (*macAudioAPI, error) { return a, nil }}
	t.Cleanup(func() {
		if len(f.strings) != 0 {
			t.Errorf("leaked %d CFStrings", len(f.strings))
		}
		if f.token != 0 {
			if _, ok := macInputs.Load(f.token); ok && f.controller.quarantined == nil {
				t.Error("callback registry leaked")
			}
			// A deliberately quarantined fixture has no actual driver. Drop its
			// synthetic entry only after assertions; production never does this.
			macInputs.Delete(f.token)
		}
	})
	return f
}

func (f *macAudioFixture) deliver(pcm []byte) {
	b := f.buffers[0]
	copy(f.backing[0], pcm)
	b.Length = uint32(len(pcm))
	macInputCallback(f.token, unsafe.Pointer(f.queue), b)
}

func TestMacAudioNativeLayouts(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("macOS supports LP64 only")
	}
	var format macAudioFormat
	var buffer macAudioBuffer
	var property macPropertyAddress
	if unsafe.Sizeof(format) != 40 || unsafe.Offsetof(format.FormatID) != 8 || unsafe.Offsetof(format.Reserved) != 36 {
		t.Fatal("AudioStreamBasicDescription ABI mismatch")
	}
	if unsafe.Sizeof(buffer) != 56 || unsafe.Offsetof(buffer.Data) != 8 || unsafe.Offsetof(buffer.Length) != 16 || unsafe.Offsetof(buffer.UserData) != 24 || unsafe.Offsetof(buffer.Packets) != 40 || unsafe.Offsetof(buffer.PacketCount) != 48 {
		t.Fatal("AudioQueueBuffer ABI mismatch")
	}
	if unsafe.Sizeof(property) != 12 {
		t.Fatal("AudioObjectPropertyAddress ABI mismatch")
	}
}

func TestMacAudioDevicesUseUIDAndFailWhenAbsent(t *testing.T) {
	f := newMacAudioFixture(t)
	devices, err := f.api.inputDevices()
	if err != nil || len(devices) != 2 || devices[0].ID != "builtin-uid" || devices[0].Label != "Built-in 麦克风" || devices[1].ID != "usb-uid" {
		t.Fatalf("roster: %v %v", devices, err)
	}
	f.ids = []uint32{30, 20, 10} // identities survive HAL reordering
	n, err := f.controller.open(context.Background(), 4, Device{ID: "usb-uid"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if f.selected != "usb-uid" {
		t.Fatal("selected the device index instead of UID")
	}
	_, _ = n.release(false)
	if _, err := f.controller.open(context.Background(), 4, Device{ID: "missing"}, false); err == nil || f.starts != 1 {
		t.Fatal("missing saved device fell back to another input")
	}
}

func TestMacAudioRecordsBoundsPartialStopAndWipes(t *testing.T) {
	f := newMacAudioFixture(t)
	n, err := f.controller.open(context.Background(), 6, Device{}, false)
	if err != nil {
		t.Fatal(err)
	}
	f.deliver([]byte{1, 0, 2, 0})
	f.partialOnStop = []byte{3, 0, 4, 0} // synchronous stop flushes partial input
	pcm, err := n.release(true)
	if err != nil || !bytes.Equal(pcm, []byte{1, 0, 2, 0, 3, 0}) {
		t.Fatalf("bounded partial PCM: %v %v", pcm, err)
	}
	if f.enqueues != 4 || f.stops != 1 || f.disposals != 1 {
		t.Fatal("callbacks requeued while stopping or cleanup was skipped")
	}
	for _, b := range f.backing {
		if !bytes.Equal(b, make([]byte, len(b))) {
			t.Fatal("native buffer retained samples")
		}
	}
	if n.pcm != nil || f.controller.busy {
		t.Fatal("capture state retained after disposal")
	}
}

func TestMacAudioErrorAndDisposalQuarantine(t *testing.T) {
	f := newMacAudioFixture(t)
	n, err := f.controller.open(context.Background(), 8, Device{}, false)
	if err != nil {
		t.Fatal(err)
	}
	f.enqueueStatus = -66687
	f.deliver([]byte{1, 0})
	select {
	case <-n.finished:
	default:
		t.Fatal("callback error did not signal owner")
	}
	f.disposeStatus = -66685
	if pcm, err := n.release(true); pcm != nil || err == nil {
		t.Fatal("native error exposed PCM")
	}
	if f.controller.quarantined != n || f.controller.busy {
		t.Fatal("failed dispose was not quarantined")
	}
	if _, ok := macInputs.Load(n.token); !ok {
		t.Fatal("callback state reclaimed before native disposal")
	}
	if !bytes.Equal(n.pcm, make([]byte, len(n.pcm))) {
		t.Fatal("quarantine retained PCM")
	}
	if _, err := f.controller.open(context.Background(), 8, Device{}, false); err == nil {
		t.Fatal("opened input after failed disposal")
	}
}

func TestMacAudioStartFailureReleasesQueue(t *testing.T) {
	f := newMacAudioFixture(t)
	f.startStatus = -66681
	if _, err := f.controller.open(context.Background(), 8, Device{}, false); err == nil {
		t.Fatal("accepted failed start")
	}
	if f.controller.busy || f.disposals != 1 {
		t.Fatal("failed start leaked queue")
	}
	f.startStatus = 0
	n, err := f.controller.open(context.Background(), 8, Device{}, false)
	if err != nil {
		t.Fatal("capture blocked after successful cleanup", err)
	}
	_, _ = n.release(false)
}

func TestMacAudioPreviewRetainsOnlyLevel(t *testing.T) {
	f := newMacAudioFixture(t)
	n, err := f.controller.open(context.Background(), 0, Device{}, true)
	if err != nil {
		t.Fatal(err)
	}
	f.deliver([]byte{0, 64, 0, 192}) // +0.5, -0.5
	if got := math.Float64frombits(n.level.Load()); math.Abs(got-0.5) > 0.0001 || n.pcm != nil {
		t.Fatalf("preview %v retained PCM", got)
	}
	if _, err := f.controller.open(context.Background(), 8, Device{}, false); err == nil {
		t.Fatal("recording overlapped preview")
	}
	pcm, err := n.release(false)
	if err != nil || pcm != nil || n.level.Load() != 0 {
		t.Fatal("preview cleanup retained audio or meter")
	}
}

func TestMacAudioCancelAndConsumeOnce(t *testing.T) {
	f := newMacAudioFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	n, err := f.controller.open(ctx, 8, Device{}, false)
	if err != nil {
		t.Fatal(err)
	}
	f.deliver([]byte{1, 0, 2, 0})
	c := newRecording(ctx)
	go n.run(c, time.Minute)
	cancel()
	c.Cancel()
	if _, err := c.Stop(); !errors.Is(err, ErrCanceled) || f.controller.busy {
		t.Fatal("cancel retained device/audio", err)
	}
}

func TestMacAudioRejectsMalformedBuffer(t *testing.T) {
	f := newMacAudioFixture(t)
	n, err := f.controller.open(context.Background(), 8, Device{}, false)
	if err != nil {
		t.Fatal(err)
	}
	f.buffers[0].Length = f.buffers[0].Capacity + 1
	macInputCallback(f.token, unsafe.Pointer(f.queue), f.buffers[0])
	if pcm, err := n.release(true); pcm != nil || err == nil {
		t.Fatal("invalid native length accepted")
	}
}

func TestMacPermissionCancelSharesPendingAndLateGrantCannotOpen(t *testing.T) {
	var callback func(bool)
	var requests atomic.Int32
	p := &macPermission{usage: func() error { return nil }, status: func() int64 { return 0 }, request: func(done func(bool)) { requests.Add(1); callback = done }}
	ctx, cancel := context.WithCancel(context.Background())
	requested := make(chan struct{})
	p.request = func(done func(bool)) { requests.Add(1); callback = done; close(requested) }
	done := make(chan error, 1)
	go func() { done <- p.authorize(ctx) }()
	<-requested
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	// A second waiter shares the still-pending dialog until its own deadline.
	if err := p.authorize(ctx2); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatal("canceled prompt allocated another native callback")
	}
	callback(true) // completion after both owners canceled; no queue can open
	p.status = func() int64 { return 3 }
	if err := p.authorize(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestMacPermissionDeniedUsageAndCanceledNeverOpen(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int64
		usage    error
		canceled bool
	}{
		{"denied", 2, nil, false}, {"restricted", 1, nil, false}, {"missing-description", 3, errors.New("missing NSMicrophoneUsageDescription"), false}, {"canceled", 3, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMacAudioFixture(t)
			p := &macPermission{usage: func() error { return tc.usage }, status: func() int64 { return tc.status }, request: func(func(bool)) { t.Fatal("unexpected permission request") }}
			f.api.permission = p.authorize
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.canceled {
				cancel()
			}
			if _, err := f.controller.open(ctx, 8, Device{}, false); err == nil || f.starts != 0 || f.controller.busy {
				t.Fatal("invalid permission state opened an input")
			}
		})
	}
}

func TestMacCanceledPermissionLateGrantNeverOpensQueue(t *testing.T) {
	f := newMacAudioFixture(t)
	var callback func(bool)
	requested := make(chan struct{})
	var granted atomic.Bool
	permission := &macPermission{
		usage: func() error { return nil },
		status: func() int64 {
			if granted.Load() {
				return 3
			}
			return 0
		},
		request: func(done func(bool)) { callback = done; close(requested) },
	}
	f.api.permission = permission.authorize
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := f.controller.open(ctx, 8, Device{}, false); done <- err }()
	<-requested
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	granted.Store(true)
	callback(true)
	if f.starts != 0 || f.token != 0 || f.controller.busy {
		t.Fatal("late consent opened a canceled input")
	}
	n, err := f.controller.open(context.Background(), 8, Device{}, false)
	if err != nil || f.starts != 1 {
		t.Fatal("new explicit action failed after consent", err)
	}
	_, _ = n.release(false)
}

func TestMacAudioPreviewLifecycleReleasesBeforeCancelReturns(t *testing.T) {
	f := newMacAudioFixture(t)
	n, err := f.controller.open(context.Background(), 0, Device{}, true)
	if err != nil {
		t.Fatal(err)
	}
	c := newRecording(context.Background())
	p := &macInputPreview{native: n, capture: c}
	f.deliver([]byte{0, 64, 0, 192})
	go n.runPreview(c)
	if p.Level() != 0.5 || p.Err() != nil {
		t.Fatal("preview did not report current level")
	}
	p.Cancel()
	p.Cancel()
	if p.Level() != 0 || p.Err() != nil || f.controller.busy || f.disposals != 1 {
		t.Fatal("preview returned before releasing input")
	}
}

func TestMacDeviceRosterEmptyAndMalformed(t *testing.T) {
	f := newMacAudioFixture(t)
	f.ids = nil
	devices, err := f.api.inputDevices()
	if err != nil || len(devices) != 0 || f.starts != 0 {
		t.Fatal("empty roster accessed microphone")
	}
	f.ids = []uint32{10, 30}
	f.uids[30] = f.uids[10]
	if _, err := f.api.inputDevices(); err == nil {
		t.Fatal("duplicate UID accepted")
	}
	f.api.propertySize = func(_ uint32, _ *macPropertyAddress, _ uint32, _ unsafe.Pointer, size *uint32) int32 {
		*size = 65540
		return 0
	}
	if _, err := f.api.inputDevices(); err == nil {
		t.Fatal("unbounded HAL list accepted")
	}
}

func TestMacAudioWAVFromNativeFixture(t *testing.T) {
	f := newMacAudioFixture(t)
	n, err := f.controller.open(context.Background(), 4, Device{}, false)
	if err != nil {
		t.Fatal(err)
	}
	f.deliver([]byte{1, 0, 2, 0, 3, 0})
	c := newRecording(context.Background())
	go n.run(c, time.Minute)
	wav, err := c.Stop()
	if err != nil || len(wav) != 48 || binary.LittleEndian.Uint32(wav[40:44]) != 4 {
		t.Fatalf("native WAV: %v", err)
	}
	if _, err := c.Stop(); !errors.Is(err, ErrStopped) {
		t.Fatal("native result consumed twice")
	}
}
