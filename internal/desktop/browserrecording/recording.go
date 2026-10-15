// Package browserrecording encodes only explicitly supplied Browser-page PNGs.
// It never obtains a display, camera, microphone, browser URL or Host credential.
package browserrecording

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxBytes           = 64 * 1024 * 1024
	MaxDuration        = 2 * time.Minute
	FrameInterval      = 125 * time.Millisecond
	MaxFrameBytes      = 8 * 1024 * 1024
	TransferChunkBytes = 256 * 1024
)

var errDurationLimit = errors.New("browser recording: duration limit reached")

type Progress struct {
	Phase        string
	Frames       int
	Elapsed      time.Duration
	LimitReached bool
}
type Artifact struct {
	MIME string
	Data []byte
}
type encoder interface {
	Start(context.Context, []byte) (string, error)
	Frame(context.Context, []byte) error
	Finish(context.Context) (Artifact, error)
	Close()
}

// Capture is called only by the native Browser Start gesture. current pins the
// original Host and navigation epoch. It must be safe to call off the UI thread.
// Every capture is consumed before the next one; Stop finalizes in memory only.
// Cancellation (including navigation/owner closure) discards the whole result.
func Capture(ctx context.Context, capture func(context.Context) ([]byte, error), current func() bool, stop <-chan struct{}, progress func(Progress) error) (Artifact, error) {
	if ctx == nil || capture == nil || current == nil {
		return Artifact{}, errors.New("browser recording: missing owner")
	}
	if ctx.Err() != nil || !current() {
		return Artifact{}, context.Canceled
	}
	e, err := newEncoder(ctx)
	if err != nil {
		return Artifact{}, err
	}
	return run(ctx, capture, current, stop, progress, e)
}

func run(ctx context.Context, capture func(context.Context) ([]byte, error), current func() bool, stop <-chan struct{}, progress func(Progress) error, e encoder) (Artifact, error) {
	defer e.Close()
	valid := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !current() {
			return context.Canceled
		}
		return nil
	}
	if err := valid(); err != nil {
		return Artifact{}, err
	}
	frame, err := capture(ctx)
	if err != nil {
		return Artifact{}, err
	}
	if err = validateFrame(frame); err != nil {
		return Artifact{}, err
	}
	if err = valid(); err != nil {
		return Artifact{}, err
	}
	mime, err := e.Start(ctx, frame)
	frame = nil
	if err != nil {
		return Artifact{}, err
	}
	if !validMIME(mime) {
		return Artifact{}, errors.New("browser recording: WebM encoding unavailable")
	}
	started := time.Now()
	p := Progress{Phase: "recording", Frames: 1}
	report := func() error {
		p.Elapsed = time.Since(started)
		if progress != nil {
			return progress(p)
		}
		return nil
	}
	if err = report(); err != nil {
		return Artifact{}, err
	}
	duration := time.NewTimer(MaxDuration)
	defer duration.Stop()
	for {
		if err = valid(); err != nil {
			return Artifact{}, err
		}
		tick := time.NewTimer(FrameInterval)
		finish := false
		select {
		case <-ctx.Done():
			tick.Stop()
			return Artifact{}, ctx.Err()
		case <-stop:
			tick.Stop()
			finish = true
		case <-duration.C:
			tick.Stop()
			finish = true
			p.LimitReached = true
		case <-tick.C:
		}
		if err = valid(); err != nil {
			return Artifact{}, err
		}
		if finish {
			break
		}
		frame, err = capture(ctx)
		if err != nil {
			return Artifact{}, err
		}
		if err = validateFrame(frame); err != nil {
			return Artifact{}, err
		}
		if err = valid(); err != nil {
			return Artifact{}, err
		}
		if err = e.Frame(ctx, frame); errors.Is(err, errDurationLimit) {
			p.LimitReached = true
			break
		} else if err != nil {
			return Artifact{}, err
		}
		frame = nil
		p.Frames++
		if err = report(); err != nil {
			return Artifact{}, err
		}
	}
	p.Phase = "stopping"
	if err = report(); err != nil {
		return Artifact{}, err
	}
	if err = valid(); err != nil {
		return Artifact{}, err
	}
	result, err := e.Finish(ctx)
	if err != nil {
		return Artifact{}, err
	}
	if err = valid(); err != nil {
		return Artifact{}, err
	}
	if result.MIME != mime {
		return Artifact{}, errors.New("browser recording: encoder format changed")
	}
	if err = result.Validate(); err != nil {
		return Artifact{}, err
	}
	return result, nil
}

func validateFrame(data []byte) error {
	if len(data) == 0 || len(data) > MaxFrameBytes {
		return errors.New("browser recording: PNG frame exceeds 8 MiB")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("browser recording: invalid PNG frame: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > 4*1024*1024 {
		return errors.New("browser recording: frame exceeds 4 megapixels")
	}
	return nil
}
func validMIME(mime string) bool {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(mime), " ", "")) {
	case "video/webm", "video/webm;codecs=vp8", "video/webm;codecs=vp9":
		return true
	}
	return false
}

// Validate checks bounded bytes, negotiated video-only MIME, EBML DocType and
// Segment signature. It is container validation, never a playback/decoder test.
func (a Artifact) Validate() error {
	bad := errors.New("browser recording: invalid or oversized WebM output")
	if !validMIME(a.MIME) || len(a.Data) < 64 || len(a.Data) > MaxBytes || !bytes.HasPrefix(a.Data, []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return bad
	}
	size, n, ok := ebmlSize(a.Data[4:])
	if !ok || size > 4096 || size > uint64(len(a.Data)-4-n) {
		return bad
	}
	end := 4 + n + int(size)
	if len(a.Data)-end < 4 || !bytes.Equal(a.Data[end:end+4], []byte{0x18, 0x53, 0x80, 0x67}) {
		return bad
	}
	header := a.Data[4+n : end]
	docType := false
	for len(header) > 0 {
		idLen := ebmlIDLength(header[0])
		if idLen == 0 || len(header) < idLen+1 {
			return bad
		}
		size, n, ok = ebmlSize(header[idLen:])
		if !ok || size > uint64(len(header)-idLen-n) {
			return bad
		}
		if idLen == 2 && header[0] == 0x42 && header[1] == 0x82 {
			if docType || !bytes.Equal(header[idLen+n:idLen+n+int(size)], []byte("webm")) {
				return bad
			}
			docType = true
		}
		header = header[idLen+n+int(size):]
	}
	if !docType {
		return bad
	}
	return nil
}
func ebmlIDLength(first byte) int {
	for n := 1; n <= 4; n++ {
		if first&(0x80>>uint(n-1)) != 0 {
			return n
		}
	}
	return 0
}
func ebmlSize(data []byte) (uint64, int, bool) {
	if len(data) == 0 {
		return 0, 0, false
	}
	n := 1
	for n <= 8 && data[0]&(0x80>>uint(n-1)) == 0 {
		n++
	}
	if n > 8 || len(data) < n {
		return 0, 0, false
	}
	value := uint64(data[0] & byte((0x80>>uint(n-1))-1))
	for i := 1; i < n; i++ {
		value = value<<8 | uint64(data[i])
	}
	return value, n, value != (uint64(1)<<uint(7*n))-1
}

// Export writes only after a native save dialog and explicit Export gesture.
// Existing files (including symlinks) are never replaced. Cancel removes only
// the new file this call created; a failed export keeps in-memory bytes for retry.
func (a Artifact) Export(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.Validate(); err != nil {
		return err
	}
	if path == "" || !strings.EqualFold(filepath.Ext(path), ".webm") {
		return errors.New("请选择新的 .webm 文件名")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("导出失败；请选择尚不存在的 .webm 文件：%w", err)
	}
	identity, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	committed := false
	defer func() {
		_ = file.Close()
		if !committed {
			if current, statErr := os.Lstat(path); statErr == nil && os.SameFile(identity, current) {
				_ = os.Remove(path)
			}
		}
	}()
	for offset := 0; offset < len(a.Data); {
		if err = ctx.Err(); err != nil {
			return err
		}
		end := min(offset+TransferChunkBytes, len(a.Data))
		n, writeErr := file.Write(a.Data[offset:end])
		if writeErr != nil {
			return writeErr
		}
		if n != end-offset {
			return errors.New("browser recording: short export write")
		}
		offset = end
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	committed = true
	return nil
}
