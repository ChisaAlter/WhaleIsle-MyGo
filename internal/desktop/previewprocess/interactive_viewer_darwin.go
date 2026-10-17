//go:build darwin && (amd64 || arm64)

package previewprocess

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/egoist/mygo/ui"
)

// The parent holds pixels only. Inline/expanded cards reuse this same Session,
// renderer and 700px viewport; changing the seat never reloads its document.
func startMacPixelSeat(ctx context.Context, s *Session, input io.Writer, output io.Reader, nonce string, doc Document, first []byte) error {
	bitmap, err := ui.DecodeBitmap(first)
	if err != nil {
		return err
	}
	commands := make(chan packet, 64)
	enqueue := func(p packet) bool {
		select {
		case <-s.Done():
			return false
		default:
		}
		if enqueueInteractiveCommand(commands, p) {
			return true
		}
		go s.Close()
		return false
	}
	s.pixels = &pixelSurface{bitmap: bitmap, width: doc.Width, enqueue: enqueue}
	s.height.Store(interactiveHeight)
	s.sendTheme = func(appearance string, vars map[string]string) error {
		if !enqueue(packet{Kind: "theme", Appearance: appearance, Variables: vars}) {
			return errors.New("macOS preview command queue closed")
		}
		return nil
	}
	go func() {
		defer s.Close()
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		suspended := false
		for {
			var command packet
			var render, apiResponse []byte
			select {
			case <-ctx.Done():
				return
			case <-s.Done():
				return
			case command = <-commands:
			case apiResponse = <-s.openUIAPIResponses():
				if !s.receivedOpenUIAPIResponse(apiResponse) {
					return
				}
				command.Kind = "api-response"
			case now := <-tick.C:
				render = s.takeOpenUIUpdate(now)
				command.Kind = "poll"
				if len(render) > 0 {
					command.Kind = "render"
				}
			}
			s.pixels.mu.Lock()
			active := time.Since(s.pixels.lastPaint) < 300*time.Millisecond
			s.pixels.mu.Unlock()
			var send bool
			command, send = inlinePixelCommand(command, active, &suspended)
			if !send {
				continue
			}
			command.Version = protocolVersion
			command.Nonce = nonce
			// Exactly one request/frame pair is in flight, including theme/release.
			watchdog := time.AfterFunc(5*time.Second, func() { _ = s.Close() })
			var writeErr error
			if len(apiResponse) > 0 {
				writeErr = sendOpenUIAPIResponse(input, nonce, apiResponse)
			} else if len(render) > 0 {
				writeErr = sendOpenUIUpdate(input, nonce, render)
			} else {
				writeErr = writePacket(input, command)
			}
			if writeErr != nil {
				watchdog.Stop()
				return
			}
			frame, err := s.receiveOpenUIFrame(output, nonce, doc.Width)
			watchdog.Stop()
			if err != nil {
				return
			}
			next, err := ui.DecodeBitmap(frame)
			if err != nil {
				return
			}
			s.pixels.mu.Lock()
			s.pixels.bitmap = next
			s.pixels.mu.Unlock()
		}
	}()
	return nil
}
