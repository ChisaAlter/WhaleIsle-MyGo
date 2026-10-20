//go:build linux && (amd64 || arm64)

package previewprocess

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/storelocale"
)

// This window contains native pixels only, never document HTML or a webview.
// A single worker owns the pipe and there is at most one outstanding frame.
func startPrivateViewer(ctx context.Context, s *Session, input io.Writer, output io.Reader, nonce string, doc Document, first []byte) error {
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
		// Losing a button release would leave the private renderer dragging.
		go s.Close()
		return false
	}
	s.sendTheme = func(appearance string, variables map[string]string) error {
		if !enqueue(packet{Kind: "theme", Appearance: appearance, Variables: variables}) {
			return errors.New("private preview input queue closed")
		}
		return nil
	}
	var viewer *mygo.Window
	if doc.Inline {
		s.pixels = &pixelSurface{bitmap: bitmap, width: doc.Width, enqueue: enqueue}
		s.height.Store(interactiveHeight)
	} else {
		viewer = mygo.NewWindow(mygo.WindowOptions{Title: storelocale.T(storelocale.KeyDesktopPreviewprocessIsolatedInteractivePreview), Width: doc.Width, Height: interactiveHeight, UseContentSize: true, DisableResize: true,
			Content: ui.View(func(c *ui.Context) {
				ui.Image(c, bitmap).Width(float32(doc.Width)).Height(interactiveHeight).Focusable().AutoFocus().TextCaret(ui.Rect{X: 0, Y: 0, W: 1, H: 20}).HandleInput(func(e ui.InputEvent) bool {
					v := interactiveInput{}
					switch e.Kind {
					case ui.InputPointerDown, ui.InputPointerUp, ui.InputPointerMove, ui.InputScroll:
						v.Kind = "pointer"
						v.X = max(0, min(doc.Width-1, int(e.X)))
						v.Y = max(0, min(interactiveHeight-1, int(e.Y)))
						if e.Kind == ui.InputPointerDown || e.Kind == ui.InputPointerUp {
							if e.Button < 0 || e.Button > 2 {
								return false
							}
							v.Button = []int{1, 3, 2}[e.Button]
							v.Down = e.Kind == ui.InputPointerDown
						}
						if e.Kind == ui.InputScroll {
							if e.DY == 0 {
								return false
							}
							v.Button = 5
							if e.DY < 0 {
								v.Button = 4
							}
						}
					case ui.InputText:
						v.Kind = "text"
						v.Text = e.Text
					case ui.InputKeyDown:
						if e.Mods != 0 {
							return false
						}
						v.Kind = "key"
						v.Key = map[ui.Key]string{ui.KeyEnter: "Return", ui.KeyEscape: "Escape", ui.KeyBackspace: "BackSpace", ui.KeyTab: "Tab", ui.KeyDelete: "Delete", ui.KeyHome: "Home", ui.KeyEnd: "End", ui.KeyPageUp: "Prior", ui.KeyPageDown: "Next", ui.KeyLeft: "Left", ui.KeyRight: "Right", ui.KeyUp: "Up", ui.KeyDown: "Down"}[e.Key]
					default:
						return false
					}
					if validateInteractiveInput(v, doc.Width) != nil {
						return false
					}
					return enqueue(packet{Kind: "poll", Input: &v})
				})
			}),
		})
		if viewer == nil || viewer.IsDestroyed() {
			return errors.New("private preview viewer could not open")
		}
		viewer.OnClosed(func() { go s.Close() })
	}
	go func() {
		defer s.Close()
		if viewer != nil {
			defer viewer.Destroy()
		}
		suspended := false
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			var command packet
			select {
			case <-ctx.Done():
				return
			case <-s.Done():
				return
			case command = <-commands:
			case <-tick.C:
				command.Kind = "poll"
			}
			if doc.Inline {
				s.pixels.mu.Lock()
				active := time.Since(s.pixels.lastPaint) < 300*time.Millisecond
				s.pixels.mu.Unlock()
				var send bool
				command, send = inlinePixelCommand(command, active, &suspended)
				if !send {
					continue
				}
			}
			command.Version = protocolVersion
			command.Nonce = nonce
			watchdog := time.AfterFunc(5*time.Second, func() { _ = s.Close() })
			if writePacket(input, command) != nil {
				watchdog.Stop()
				return
			}
			frame, err := receiveFrame(output, nonce, doc.Width)
			watchdog.Stop()
			if err != nil {
				return
			}
			next, err := ui.DecodeBitmap(frame)
			if err != nil {
				return
			}
			if doc.Inline {
				s.pixels.mu.Lock()
				s.pixels.bitmap = next
				s.pixels.mu.Unlock()
				continue
			}
			// Backpressure includes UI publication, so slow/minimized viewers cannot
			// accumulate decoded frames in the application dispatch queue.
			published := make(chan struct{})
			viewer.Update(func() { bitmap = next; close(published) })
			select {
			case <-published:
			case <-ctx.Done():
				return
			case <-s.Done():
				return
			}
		}
	}()
	return nil
}
