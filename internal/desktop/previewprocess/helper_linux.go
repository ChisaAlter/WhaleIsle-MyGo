//go:build linux && (amd64 || arm64)

package previewprocess

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/htmlpreview"
)

// RunHelper runs only the trusted namespace-contained companion. stdin/stdout
// carry one bounded document plus capture or typed frame/input messages, never RPC.
func RunHelper(args []string, input io.Reader, output io.Writer) error {
	if len(args) != 3 || args[0] != "--namespace-capture-v1" {
		return errors.New("Linux capture helper requires a namespace parent handshake")
	}
	nonce, parent := args[1], args[2]
	if b, err := hex.DecodeString(nonce); err != nil || len(b) != 32 {
		return errors.New("invalid helper nonce")
	}
	current, err := namespaceIdentity()
	if err != nil || !isolatedNamespace(parent, current) {
		return errors.New("capture helper namespaces are not isolated from the parent")
	}
	// Never use a host X/Wayland socket. The display lives in the same private
	// mount/network/PID namespaces as the renderer, with no listening TCP port.
	if err := os.Chmod("/run/preview", 0700); err != nil {
		return err
	}
	display := exec.Command("/usr/bin/Xvfb", ":99", "-screen", "0", "1600x4000x24", "-nolisten", "tcp", "-noreset")
	display.Stdout, display.Stderr = os.Stderr, os.Stderr
	if err := display.Start(); err != nil {
		return err
	}
	defer func() { _ = display.Process.Kill(); _ = display.Wait() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if st, err := os.Stat("/tmp/.X11-unix/X99"); err == nil && st.Mode()&os.ModeSocket != 0 {
			break
		}
		if time.Now().After(deadline) {
			return errors.New("private preview display did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := os.Setenv("DISPLAY", ":99"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result := make(chan error, 1)
	mygo.App.WhenReady(func() {
		// The ordinary in-process IsolatedPreview blank encoder remains unchanged.
		// This window exists only behind the outer OS sandbox, with no host profile,
		// native bindings, app protocols or privileged services registered.
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Isolated capture", Width: 900, Height: 800, UseContentSize: true, Hidden: true, SkipTaskbar: true, Frameless: true, DisableResize: true,
			Page: mygo.PageOptions{DisableBindings: true, DevTools: mygo.DevToolsDisabled}})
		p := w.Page()
		p.SetWindowOpenHandler(func(mygo.WindowOpenRequest) *mygo.WindowOptions { return nil })
		p.SetPermissionHandler(func(mygo.PermissionRequest) bool { return false })
		p.OnWillDownload(func(e *mygo.DownloadEvent) { e.PreventDefault() })
		p.OnWillNavigate(func(e *mygo.NavigateEvent) {
			if e.URL != "about:blank" && e.URL != "about:srcdoc" {
				e.PreventDefault()
			}
		})
		p.OnRenderProcessGone(func(string) { cancel() })
		go func() {
			err := func() error {
				if err := writePacket(output, packet{Version: protocolVersion, Kind: "ready", Nonce: nonce, SID: parent}); err != nil {
					return err
				}
				doc, err := receiveDocument(input, nonce)
				if err != nil {
					return err
				}
				if doc.Inline {
					return errors.New("namespace helper does not support inline native embedding")
				}
				if !doc.capture {
					return servePrivateFrames(ctx, w, input, output, nonce, doc)
				}
				startup, stop := context.WithTimeout(ctx, 25*time.Second)
				defer stop()
				capture, err := htmlpreview.CaptureProcessDocument(startup, w, doc.HTML, doc.Width, doc.Appearance)
				if err != nil {
					return err
				}
				if err = writePacket(output, packet{Version: protocolVersion, Kind: "loaded", Nonce: nonce}); err != nil {
					return err
				}
				return sendCapture(output, nonce, capture)
			}()
			result <- err
			// Parent closes the pipe after validating the result. Retain the contained
			// helper until then so normal process exit cannot truncate a pending read.
			if err == nil {
				var b [1]byte
				_, _ = input.Read(b[:])
			}
			w.Destroy()
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	default:
		return fmt.Errorf("preview exited before producing a result")
	}
}
