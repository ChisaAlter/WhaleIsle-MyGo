//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"golang.org/x/sys/windows"
	"whaleisle/internal/desktop/htmlpreview"
)

// RunHelper must be called from the dedicated executable's main goroutine.
// It refuses to initialize MyGo outside an already-contained process. It
// neither creates a profile nor decides whether a document is authorized.
func RunHelper(args []string, input io.Reader, output io.Writer) error {
	if len(args) != 3 || args[0] != "--appcontainer-helper-v1" {
		return errors.New("preview helper requires a parent handshake")
	}
	nonce, sid := args[1], args[2]
	if decoded, err := hex.DecodeString(nonce); err != nil || len(decoded) != 32 {
		return errors.New("invalid helper nonce")
	}
	if err := verifyCurrentToken(sid); err != nil {
		return err
	}
	if err := verifyInJob(windows.CurrentProcess(), 0); err != nil {
		return err
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	var returned uint32
	if err := windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), &returned); err != nil {
		return err
	}
	flags := limits.BasicLimitInformation.LimitFlags
	if flags&windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE == 0 || flags&(windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK|windows.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK) != 0 {
		return errors.New("helper is not in a non-breakaway lifetime-bound job")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result := make(chan error, 1)
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title: "Isolated interactive HTML preview", Width: 900, Height: 700,
			UseContentSize: true, Hidden: true,
			Page: mygo.PageOptions{IsolatedPreview: true, DisableBindings: true, DevTools: mygo.DevToolsDisabled},
		})
		w.OnClosed(cancel)
		w.Page().OnRenderProcessGone(func(string) { cancel() })
		go func() {
			err := serveDocument(ctx, w, input, output, nonce, sid)
			result <- err
			w.Destroy()
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	cancel()
	select {
	case err := <-result:
		return err
	default:
		return nil // Parent observes process exit and destroys the whole job.
	}
}

func loadHelperBlank(ctx context.Context, w *mygo.Window) error {
	done := make(chan error, 1)
	deliver := func(err error) {
		select {
		case done <- err:
		default:
		}
	}
	offLoad := w.Page().OnDidFinishLoad(func() { deliver(nil) })
	defer offLoad()
	offFail := w.Page().OnDidFailLoad(func(err *mygo.LoadError) { deliver(err) })
	defer offFail()
	offGone := w.Page().OnRenderProcessGone(func(reason string) { deliver(fmt.Errorf("preview renderer exited: %s", reason)) })
	defer offGone()
	if err := w.Page().LoadURL("about:blank"); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func serveDocument(ctx context.Context, w *mygo.Window, input io.Reader, output io.Writer, nonce, sid string) error {
	startup, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if err := loadHelperBlank(startup, w); err != nil {
		return err
	}
	if err := verifyCurrentToken(sid); err != nil {
		return err
	}
	pid, err := w.Page().BrowserProcessID()
	if err != nil {
		return err
	}
	// No document is in memory yet. This identity is sent over the inherited
	// pipe, not a page bridge. Only the parent's OS query grants the next phase.
	if err := writePacket(output, packet{Version: protocolVersion, Kind: "ready", Nonce: nonce, OpenUI: true, OpenUIUpdate: true, OpenUIAPI: true, OpenUIAPIBytes: maxHTMLBytes, SID: sid, HelperPID: uint32(os.Getpid()), BrowserPID: pid}); err != nil {
		return err
	}
	doc, err := receiveDocument(input, nonce)
	if err != nil {
		return err
	}
	if err := startup.Err(); err != nil {
		return err
	}
	if err := verifyCurrentToken(sid); err != nil {
		return err
	}
	currentPID, err := w.Page().BrowserProcessID()
	if err != nil || currentPID != pid {
		return errors.New("preview browser changed before document load")
	}
	if doc.capture {
		result, err := htmlpreview.CaptureProcessDocument(startup, w, doc.HTML, doc.Width, doc.Appearance)
		if err != nil {
			return err
		}
		currentPID, err = w.Page().BrowserProcessID()
		if err != nil || currentPID != pid {
			return errors.New("preview browser changed during capture")
		}
		if err := writePacket(output, packet{Version: protocolVersion, Kind: "loaded", Nonce: nonce, BrowserPID: pid}); err != nil {
			return err
		}
		if err := sendCapture(output, nonce, result); err != nil {
			return err
		}
		// Keep the browser tree alive until the parent validates and closes it.
		var b [1]byte
		_, _ = input.Read(b[:])
		return nil
	}
	shell, err := documentShell(doc)
	if err != nil {
		return err
	}
	w.SetContentSize(doc.Width, 700)
	encoded, err := json.Marshal(shell)
	if err != nil {
		return err
	}
	if doc.openUIShell {
		// The mode comes only from the nonce-checked parent document packet,
		// after token/job/browser identity verification, never from HTML or JS.
		// Keep ordinary HTML on the original single-srcdoc admission policy.
		if err := w.Page().EnableIsolatedOpenUIFrames(); err != nil {
			return err
		}
	}
	if _, err := w.Page().EvalContext(startup, "document.open();document.write("+string(encoded)+");document.close();true"); err != nil {
		return err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	openUIFrameSupportSet := !doc.openUIShell
	for {
		if !openUIFrameSupportSet {
			known, htmlView, err := w.Page().IsolatedOpenUIFrameSupport()
			if err != nil {
				return err
			}
			if known {
				script := "(()=>{if(window.__whaleOpenUIHtmlViewSupport(false)!==true)throw new Error('OpenUI HtmlView support rejected');return true;})()"
				if htmlView {
					script = "(()=>{if(window.__whaleOpenUIHtmlViewSupport(true)!==true)throw new Error('OpenUI HtmlView support rejected');return true;})()"
				}
				if _, err := w.Page().EvalContext(startup, script); err != nil {
					return err
				}
				openUIFrameSupportSet = true
			}
		}
		value, err := w.Page().EvalContext(startup, "({ready:window.__previewState?.ready===true,error:window.__previewState?.error||''})")
		if err != nil {
			return err
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var state struct {
			Ready bool
			Error string
		}
		if err := json.Unmarshal(raw, &state); err != nil {
			return err
		}
		if state.Error != "" {
			return errors.New("interactive document failed to initialize")
		}
		if state.Ready && openUIFrameSupportSet {
			break
		}
		select {
		case <-startup.Done():
			return startup.Err()
		case <-ticker.C:
		}
	}
	currentPID, err = w.Page().BrowserProcessID()
	if err != nil || currentPID != pid {
		return errors.New("preview browser changed during document load")
	}
	if err := writePacket(output, packet{Version: protocolVersion, Kind: "loaded", Nonce: nonce, BrowserPID: pid, WindowHandle: uint64(w.NativeHandle())}); err != nil {
		return err
	}
	if !doc.Inline {
		w.Show()
	}
	// Observe bounded size and, only in OpenUI mode, drain the trusted shell
	// queue. No event can choose a native operation or an IPC command.
	go func() {
		// Fail closed on a stalled/invalid shell. There is no page-to-native bridge.
		defer func() { w.Destroy(); mygo.App.Quit() }()
		tick := time.NewTicker(150 * time.Millisecond)
		defer tick.Stop()
		last := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				pollCtx, stopPoll := context.WithTimeout(ctx, 2*time.Second)
				if doc.openUIShell {
					value, e := w.Page().EvalContext(pollCtx, openUIDrainScript)
					if e != nil {
						stopPoll()
						return
					}
					batch, e := decodeOpenUIBatch(value)
					if e != nil {
						stopPoll()
						return
					}
					for _, event := range batch {
						if sendOpenUIEvent(output, nonce, event) != nil {
							stopPoll()
							return
						}
					}
				}
				value, err := w.Page().EvalContext(pollCtx, "Math.max(80,Math.min(2000,Math.ceil(window.__previewState?.height||360)))")
				stopPoll()
				if err != nil {
					return
				}
				n, ok := value.(float64)
				if !ok {
					continue
				}
				height := int(n)
				if height != last {
					if writePacket(output, packet{Version: protocolVersion, Kind: "size", Nonce: nonce, Height: height}) != nil {
						return
					}
					last = height
				}
			}
		}
	}()
	// Only trusted parent theme updates are accepted after the one document.
	// This pipe is never exposed to the sandboxed page.
	for {
		p, err := readPacket(input)
		if err != nil {
			return err
		}
		if p.Kind == "api-response" {
			if !doc.openUIShell {
				return errors.New("ordinary HTML cannot receive OpenUI API responses")
			}
			raw, e := receiveOpenUIAPIResponse(input, p, nonce)
			if e != nil {
				return e
			}
			script, e := openUIAPIResponseScript(raw)
			if e != nil {
				return e
			}
			responseCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			_, e = w.Page().EvalContext(responseCtx, script)
			stop()
			if e != nil {
				return e
			}
			continue
		}
		if p.Kind == "render" {
			if !doc.openUIShell {
				return errors.New("ordinary HTML cannot receive OpenUI updates")
			}
			raw, e := receiveOpenUIUpdate(input, p, nonce)
			if e != nil {
				return e
			}
			script, e := openUIUpdateScript(raw)
			if e != nil {
				return e
			}
			renderCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			_, e = w.Page().EvalContext(renderCtx, script)
			stop()
			if e != nil {
				return e
			}
			continue
		}
		if err = expect(p, "theme", nonce); err != nil {
			return err
		}
		script, err := htmlpreview.ProcessThemeTokensScript(p.Appearance, p.Variables)
		if err != nil {
			return err
		}
		themeCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		_, err = w.Page().EvalContext(themeCtx, script)
		stop()
		if err != nil {
			return err
		}
	}
}
