package browserrecording

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Synthetic protocol fixture only. These bytes assert container gating; they
// are not a playable recording and this test never constructs a native window.
func syntheticWebM() []byte {
	b := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x87, 0x42, 0x82, 0x84, 'w', 'e', 'b', 'm', 0x18, 0x53, 0x80, 0x67, 0xff}
	return append(b, make([]byte, 80)...)
}
func TestSyntheticRecordingProtocol(t *testing.T) {
	good := Artifact{MIME: "video/webm;codecs=vp8", Data: syntheticWebM()}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Artifact{{MIME: "video/mp4", Data: good.Data}, {MIME: "video/webm", Data: []byte("PNG frames")}, {MIME: "video/webm", Data: append([]byte(nil), good.Data[:12]...)}, {MIME: "video/webm;codecs=unknown", Data: good.Data}} {
		if bad.Validate() == nil {
			t.Fatalf("accepted invalid video %q", bad.MIME)
		}
	}
	corrupt := append([]byte(nil), good.Data...)
	corrupt[8] = 'x'
	if (Artifact{MIME: good.MIME, Data: corrupt}).Validate() == nil {
		t.Fatal("accepted wrong EBML doctype")
	}
	var pngBuffer bytes.Buffer
	if err := png.Encode(&pngBuffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	frame := pngBuffer.Bytes()
	if err := validateFrame(frame); err != nil {
		t.Fatal(err)
	}
	if validateFrame(good.Data) == nil {
		t.Fatal("accepted non-PNG input frame")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &fixtureEncoder{artifact: good}
	if _, err := run(ctx, func(context.Context) ([]byte, error) { t.Fatal("captured after cancel"); return nil, nil }, func() bool { return true }, make(chan struct{}), nil, e); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	if !e.closed || e.started {
		t.Fatal("cancel did not close unused encoder")
	}
	stop := make(chan struct{})
	count := 0
	e = &fixtureEncoder{artifact: good}
	got, err := run(context.Background(), func(context.Context) ([]byte, error) { count++; return frame, nil }, func() bool { return true }, stop, func(p Progress) error {
		if p.Phase == "recording" {
			close(stop)
		}
		return nil
	}, e)
	if err != nil || count != 1 || !e.started || !e.finished || !e.closed || !bytes.Equal(got.Data, good.Data) {
		t.Fatalf("stop lifecycle: captures=%d encoder=%+v err=%v", count, e, err)
	}
	// Navigation invalidation at finalization discards even a valid encoded result.
	current := true
	e = &fixtureEncoder{artifact: good, onFinish: func() { current = false }}
	stop = make(chan struct{})
	close(stop)
	if _, err = run(context.Background(), func(context.Context) ([]byte, error) { return frame, nil }, func() bool { return current }, stop, nil, e); !errors.Is(err, context.Canceled) {
		t.Fatalf("stale source = %v", err)
	}
	// An encoder-side duration deadline (while native capture was pending)
	// finalizes once and never asks for a third frame.
	count = 0
	limited := false
	e = &fixtureEncoder{artifact: good, frameError: errDurationLimit}
	got, err = run(context.Background(), func(context.Context) ([]byte, error) { count++; return frame, nil }, func() bool { return true }, make(chan struct{}), func(p Progress) error { limited = p.LimitReached; return nil }, e)
	if err != nil || count != 2 || !limited || !e.finished || len(got.Data) == 0 {
		t.Fatalf("encoder duration limit: %d %v %v", count, limited, err)
	}
	current = true
	e = &fixtureEncoder{artifact: good}
	_, err = run(context.Background(), func(context.Context) ([]byte, error) { current = false; return frame, nil }, func() bool { return current }, make(chan struct{}), nil, e)
	if !errors.Is(err, context.Canceled) || e.started {
		t.Fatal("navigation during capture reached encoder")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.webm")
	if err = good.Export(ctx, path); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled export = %v", err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("canceled export created a file")
	}
	if err = good.Export(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if err = good.Export(context.Background(), path); err == nil {
		t.Fatal("overwrote existing export")
	}
	gotBytes, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(gotBytes, good.Data) {
		t.Fatal("export bytes changed")
	}
}

type fixtureEncoder struct {
	artifact                  Artifact
	started, finished, closed bool
	frameError                error
	onFinish                  func()
}

func (e *fixtureEncoder) Start(context.Context, []byte) (string, error) {
	e.started = true
	return e.artifact.MIME, nil
}
func (e *fixtureEncoder) Frame(context.Context, []byte) error {
	if e.frameError != nil {
		return e.frameError
	}
	return errors.New("unexpected extra frame")
}
func (e *fixtureEncoder) Finish(context.Context) (Artifact, error) {
	e.finished = true
	if e.onFinish != nil {
		e.onFinish()
	}
	return e.artifact, nil
}
func (e *fixtureEncoder) Close() { e.closed = true }
