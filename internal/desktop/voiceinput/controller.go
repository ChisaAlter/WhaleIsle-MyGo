// Package voiceinput owns one explicit microphone-to-draft operation. Capture,
// Host calls and UI dispatch are supplied by the desktop, not by a web page.
package voiceinput

import (
	"context"
	"sync"
	"time"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/store/storelocale"
)

type Transcribe func(context.Context, []byte) (string, error)
type Options struct {
	Prepare  func(context.Context) (voicecapture.Limits, Transcribe, error)
	Capture  func(context.Context, voicecapture.Limits) (voicecapture.Capture, error)
	Valid    func() bool
	Dispatch func(func())
	Insert   func(string) bool
	Changed  func()
}
type Status struct{ Phase, Message, Pending string }

func (s Status) Busy() bool {
	return s.Phase == "requesting" || s.Phase == "recording" || s.Phase == "transcribing"
}

type Controller struct {
	mu         sync.Mutex
	opts       Options
	status     Status
	serial     uint64
	cancel     context.CancelFunc
	capture    voicecapture.Capture
	timer      *time.Timer
	transcribe Transcribe
	ctx        context.Context
}

func New(o Options) *Controller        { return &Controller{opts: o} }
func (c *Controller) Snapshot() Status { c.mu.Lock(); defer c.mu.Unlock(); return c.status }
func (c *Controller) notify() {
	if c.opts.Changed != nil {
		c.opts.Changed()
	}
}
func (c *Controller) valid() bool { return c.opts.Valid == nil || c.opts.Valid() }
func (c *Controller) Start() {
	c.mu.Lock()
	if c.status.Busy() || c.status.Pending != "" {
		c.mu.Unlock()
		return
	}
	c.serial++
	serial := c.serial
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel
	c.status = Status{Phase: "requesting", Message: storelocale.T(storelocale.KeyDesktopVoiceinputOpeningMicrophone)}
	c.mu.Unlock()
	c.notify()
	go func() {
		limits, transcribe, err := c.opts.Prepare(ctx)
		if err != nil {
			c.complete(serial, "", err.Error())
			return
		}
		if ctx.Err() != nil || !c.valid() {
			c.cancelSerial(serial)
			return
		}
		capture, err := c.opts.Capture(ctx, limits)
		if err != nil {
			c.complete(serial, "", storelocale.T(storelocale.KeyDesktopVoiceinputMicrophoneUnavailable))
			return
		}
		c.mu.Lock()
		if serial != c.serial || ctx.Err() != nil {
			c.mu.Unlock()
			capture.Cancel()
			return
		}
		c.capture = capture
		c.transcribe = transcribe
		c.status = Status{Phase: "recording", Message: storelocale.T(storelocale.KeyDesktopVoiceinputRecording)}
		c.timer = time.AfterFunc(limits.MaxDuration, func() { c.stop(serial) })
		c.mu.Unlock()
		c.notify()
	}()
}
func (c *Controller) Stop() { c.mu.Lock(); serial := c.serial; c.mu.Unlock(); c.stop(serial) }
func (c *Controller) stop(serial uint64) {
	c.mu.Lock()
	if serial != c.serial || c.status.Phase != "recording" {
		c.mu.Unlock()
		return
	}
	capture, transcribe, ctx := c.capture, c.transcribe, c.ctx
	c.capture = nil
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.status = Status{Phase: "transcribing", Message: storelocale.T(storelocale.KeyDesktopVoiceinputTranscribing)}
	c.mu.Unlock()
	c.notify()
	go func() {
		defer capture.Cancel()
		raw, err := capture.Stop()
		if err != nil {
			c.complete(serial, "", storelocale.T(storelocale.KeyDesktopVoiceinputRecordingFailed))
			return
		}
		defer clear(raw)
		if ctx.Err() != nil || !c.valid() {
			c.cancelSerial(serial)
			return
		}
		text, err := transcribe(ctx, raw)
		if err != nil {
			c.complete(serial, "", storelocale.T(storelocale.KeyStoreSessionTranscriptionFailed))
			return
		}
		c.complete(serial, text, "")
	}()
}
func (c *Controller) complete(serial uint64, text, message string) {
	c.opts.Dispatch(func() {
		c.mu.Lock()
		if serial != c.serial {
			c.mu.Unlock()
			return
		}
		cancel := c.cancel
		c.cancel = nil
		c.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if !c.valid() {
			c.Cancel()
			return
		}
		inserted := message == "" && text != "" && c.opts.Insert(text)
		c.mu.Lock()
		if serial != c.serial {
			c.mu.Unlock()
			return
		}
		if inserted {
			c.status = Status{}
		} else if text != "" {
			c.status = Status{Phase: "feedback", Pending: text, Message: storelocale.T(storelocale.KeyDesktopVoiceinputDraftChangedInsertOrDiscard)}
		} else {
			c.status = Status{Phase: "feedback", Message: message}
		}
		c.mu.Unlock()
		c.notify()
	})
}
func (c *Controller) InsertPending(insert func(string) bool) {
	c.mu.Lock()
	text, serial := c.status.Pending, c.serial
	c.mu.Unlock()
	if text == "" || !c.valid() || !insert(text) {
		return
	}
	c.mu.Lock()
	if serial == c.serial {
		c.status = Status{}
	}
	c.mu.Unlock()
	c.notify()
}

// Cancel invalidates queued grants/results before releasing hardware. It never
// inserts text, submits a prompt or saves raw audio to disk.
func (c *Controller) Cancel() { c.cancelSerial(0) }
func (c *Controller) cancelSerial(expected uint64) {
	c.mu.Lock()
	if expected != 0 && c.serial != expected {
		c.mu.Unlock()
		return
	}
	c.serial++
	cancel, capture := c.cancel, c.capture
	c.cancel = nil
	c.capture = nil
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.status = Status{}
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if capture != nil {
		go capture.Cancel()
	}
	c.notify()
}
