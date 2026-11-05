// Package voicecapture records a bounded microphone clip in memory. Callers
// must call Start only in response to an explicit user recording action, and
// must call Stop or Cancel when the owning UI/session is dismissed.
package voicecapture

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sampleRate      = 16000
	blockAlign      = 2 // mono, signed PCM16
	bytesPerSecond  = sampleRate * blockAlign
	wavHeaderBytes  = 44
	hardMaxBytes    = 8 << 20
	hardMaxDuration = 2 * time.Minute
)

var (
	ErrUnsupported = errors.New("microphone recording requires Windows or macOS")
	ErrCanceled    = errors.New("microphone recording was canceled")
	ErrStopped     = errors.New("microphone recording has already been consumed")
	ErrNoAudio     = errors.New("no microphone audio was recorded")
)

// Limits bounds both acquisition time and the complete returned WAV size.
// Both fields must be positive. Larger values are capped at 8 MiB and two
// minutes; byte counts are rounded down to complete PCM16 samples.
type Limits struct {
	MaxBytes    int
	MaxDuration time.Duration
}

// Capture owns one recording. Stop releases the input device and returns one
// complete PCM16, mono, 16000 Hz WAV, in memory only. Stop consumes the result;
// subsequent calls return ErrStopped. Cancel is idempotent, releases the input
// device, and discards any unconsumed audio. Stop and Cancel may run concurrently.
// Reaching a limit releases the device and retains only bounded PCM until Stop,
// Cancel, or context cancellation. Callers must eventually Stop or Cancel.
type Capture interface {
	Stop() ([]byte, error)
	Cancel()
}

// LiveCapture exposes actual input amplitude and native completion without
// changing the existing Stop/Cancel acquisition contract.
type LiveCapture interface {
	Capture
	Level() float64
	Done() <-chan struct{}
}

func pcmLevel(pcm []byte) float64 {
	var sum float64
	n := len(pcm) / 2
	for i := 0; i < n; i++ {
		x := float64(int16(binary.LittleEndian.Uint16(pcm[i*2:]))) / 32768
		sum += x * x
	}
	if n == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(n))
}
func (c *recording) Level() float64        { return math.Float64frombits(c.level.Load()) }
func (c *recording) Done() <-chan struct{} { return c.done }

func boundedLimits(limits Limits) (int, time.Duration, error) {
	if limits.MaxBytes < wavHeaderBytes+blockAlign || limits.MaxDuration <= 0 {
		return 0, 0, errors.New("recording limits must allow a WAV header, at least one PCM16 sample, and a positive duration")
	}
	maxBytes := min(limits.MaxBytes, hardMaxBytes)
	duration := min(limits.MaxDuration, hardMaxDuration)
	// Multiplication is safe after clamping duration to two minutes.
	frames := int(duration * sampleRate / time.Second)
	pcmBytes := min(maxBytes-wavHeaderBytes, frames*blockAlign)
	pcmBytes -= pcmBytes % blockAlign
	if pcmBytes == 0 {
		return 0, 0, errors.New("recording duration must allow at least one PCM16 sample")
	}
	// A small byte cap also shortens device acquisition, not just the result.
	duration = time.Duration(pcmBytes/blockAlign) * time.Second / sampleRate
	return pcmBytes, duration, nil
}

// encodeWAV always creates a canonical 44-byte PCM WAV header. It never writes
// a file, uploads audio, or invokes a transcription provider.
func encodeWAV(pcm []byte) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, ErrNoAudio
	}
	if len(pcm)%blockAlign != 0 || len(pcm) > hardMaxBytes-wavHeaderBytes {
		return nil, fmt.Errorf("invalid PCM16 buffer length: %d", len(pcm))
	}
	wav := make([]byte, wavHeaderBytes+len(pcm))
	copy(wav[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(len(wav)-8))
	copy(wav[8:12], "WAVE")
	copy(wav[12:16], "fmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)
	binary.LittleEndian.PutUint16(wav[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(wav[22:24], 1) // mono
	binary.LittleEndian.PutUint32(wav[24:28], sampleRate)
	binary.LittleEndian.PutUint32(wav[28:32], bytesPerSecond)
	binary.LittleEndian.PutUint16(wav[32:34], blockAlign)
	binary.LittleEndian.PutUint16(wav[34:36], 16)
	copy(wav[36:40], "data")
	binary.LittleEndian.PutUint32(wav[40:44], uint32(len(pcm)))
	copy(wav[wavHeaderBytes:], pcm)
	return wav, nil
}

type recording struct {
	level      atomic.Uint64
	ctx        context.Context
	stop       chan struct{}
	cancel     chan struct{}
	done       chan struct{}
	stopOnce   sync.Once
	cancelOnce sync.Once

	mu        sync.Mutex
	pcm       []byte
	err       error
	canceled  bool
	consumed  bool
	stopWatch func() bool
}

func newRecording(ctx context.Context) *recording {
	c := &recording{
		ctx: ctx, stop: make(chan struct{}), cancel: make(chan struct{}), done: make(chan struct{}),
	}
	// Keep this watcher after the native device reaches a limit, so canceling
	// the owning session also discards a clip awaiting its explicit Stop call.
	c.stopWatch = context.AfterFunc(ctx, c.Cancel)
	return c
}

func (c *recording) Stop() ([]byte, error) {
	c.stopOnce.Do(func() { close(c.stop) })
	<-c.done
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopWatch != nil {
		c.stopWatch()
	}
	if c.canceled || c.ctx.Err() != nil {
		clear(c.pcm)
		c.pcm = nil
		return nil, errors.Join(ErrCanceled, c.ctx.Err(), c.err)
	}
	if c.consumed {
		return nil, ErrStopped
	}
	c.consumed = true
	if c.err != nil {
		return nil, c.err
	}
	wav, err := encodeWAV(c.pcm)
	clear(c.pcm)
	c.pcm = nil
	return wav, err
}

func (c *recording) Cancel() {
	c.mu.Lock()
	c.canceled = true
	clear(c.pcm)
	c.pcm = nil
	c.mu.Unlock()
	c.cancelOnce.Do(func() { close(c.cancel) })
	<-c.done
	// The native owner checks canceled before publishing its result. Waiting
	// here guarantees that a successful device cleanup precedes our return.
	if c.stopWatch != nil {
		c.stopWatch()
	}
}

// finish is called only after the native owner has released its device and
// buffers (or quarantined them on an unrecoverable driver cleanup failure).
func (c *recording) finish(pcm []byte, err error) {
	c.mu.Lock()
	if c.canceled || c.ctx.Err() != nil || err != nil {
		clear(pcm)
	} else {
		c.pcm = pcm
	}
	c.err = err
	c.level.Store(0)
	c.mu.Unlock()
	close(c.done)
}
