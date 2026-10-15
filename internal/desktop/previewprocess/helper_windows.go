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
	if len(args) != 3 || args[0] != "--appcontainer-helper-v1" { return errors.New("preview helper requires a parent handshake") }
	nonce, sid := args[1], args[2]
	if decoded, err := hex.DecodeString(nonce); err != nil || len(decoded) != 32 { return errors.New("invalid helper nonce") }
	if err := verifyCurrentToken(sid); err != nil { return err }
	if err := verifyInJob(windows.CurrentProcess(), 0); err != nil { return err }
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	var returned uint32
	if err := windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), &returned); err != nil { return err }
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
	if err := mygo.App.Run(); err != nil { return err }
	cancel()
	select {
	case err := <-result: return err
	default: return nil // Parent observes process exit and destroys the whole job.
	}
}

func loadHelperBlank(ctx context.Context, w *mygo.Window) error {
	done := make(chan error, 1)
	deliver := func(err error) { select { case done <- err: default: } }
	offLoad := w.Page().OnDidFinishLoad(func() { deliver(nil) })
	defer offLoad()
	offFail := w.Page().OnDidFailLoad(func(err *mygo.LoadError) { deliver(err) })
	defer offFail()
	offGone := w.Page().OnRenderProcessGone(func(reason string) { deliver(fmt.Errorf("preview renderer exited: %s", reason)) })
	defer offGone()
	if err := w.Page().LoadURL("about:blank"); err != nil { return err }
	select { case err := <-done: return err; case <-ctx.Done(): return ctx.Err() }
}

func serveDocument(ctx context.Context, w *mygo.Window, input io.Reader, output io.Writer, nonce, sid string) error {
	startup, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if err := loadHelperBlank(startup, w); err != nil { return err }
	if err := verifyCurrentToken(sid); err != nil { return err }
	pid, err := w.Page().BrowserProcessID()
	if err != nil { return err }
	// No document is in memory yet. This identity is sent over the inherited
	// pipe, not a page bridge. Only the parent's OS query grants the next phase.
	if err := writePacket(output, packet{Version: protocolVersion, Kind: "ready", Nonce: nonce, SID: sid, HelperPID: uint32(os.Getpid()), BrowserPID: pid}); err != nil { return err }
	doc, err := receiveDocument(input, nonce)
	if err != nil { return err }
	if err := startup.Err(); err != nil { return err }
	if err := verifyCurrentToken(sid); err != nil { return err }
	currentPID, err := w.Page().BrowserProcessID()
	if err != nil || currentPID != pid { return errors.New("preview browser changed before document load") }
	shell, err := htmlpreview.ProcessDocumentShell(doc.HTML, doc.Width, doc.Appearance)
	if err != nil { return err }
	w.SetContentSize(doc.Width, 700)
	encoded, err := json.Marshal(shell)
	if err != nil { return err }
	if _, err := w.Page().EvalContext(startup, "document.open();document.write("+string(encoded)+");document.close();true"); err != nil { return err }
	ticker := time.NewTicker(25*time.Millisecond)
	defer ticker.Stop()
	for {
		value, err := w.Page().EvalContext(startup, "({ready:window.__previewState?.ready===true,error:window.__previewState?.error||''})")
		if err != nil { return err }
		raw, err := json.Marshal(value)
		if err != nil { return err }
		var state struct { Ready bool; Error string }
		if err := json.Unmarshal(raw, &state); err != nil { return err }
		if state.Error != "" { return errors.New("interactive document failed to initialize") }
		if state.Ready { break }
		select { case <-startup.Done(): return startup.Err(); case <-ticker.C: }
	}
	currentPID, err = w.Page().BrowserProcessID()
	if err != nil || currentPID != pid { return errors.New("preview browser changed during document load") }
	if err := writePacket(output, packet{Version: protocolVersion, Kind: "loaded", Nonce: nonce, BrowserPID: pid}); err != nil { return err }
	w.Show()
	// The protocol accepts exactly one snapshot. EOF (parent exit/cancellation)
	// and any extra data close this window; no commands or native operations
	// can be requested by the HTML or by a subsequent protocol packet.
	parentGone := make(chan struct{})
	go func() { var b [1]byte; _, _ = input.Read(b[:]); close(parentGone) }()
	select { case <-ctx.Done(): return ctx.Err(); case <-parentGone: return nil }
}
