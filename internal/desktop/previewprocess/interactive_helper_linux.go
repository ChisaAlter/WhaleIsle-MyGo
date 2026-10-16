//go:build linux && (amd64 || arm64)

package previewprocess

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/htmlpreview"
)

// The target is discovered inside the private display. No host window ID,
// shell command, path, clipboard, or display address is accepted over IPC.
func privateInputWindow(ctx context.Context) (string, error) {
	c, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	out, err := exec.CommandContext(c, "/usr/bin/xdotool", "search", "--onlyvisible", "--pid", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return "", err
	}
	ids := strings.Fields(string(out))
	if len(ids) != 1 {
		return "", errors.New("private preview window is ambiguous")
	}
	if _, err := strconv.ParseUint(ids[0], 10, 64); err != nil {
		return "", err
	}
	return ids[0], nil
}
func applyPrivateInput(ctx context.Context, id string, v interactiveInput, width int) error {
	if err := validateInteractiveInput(v, width); err != nil {
		return err
	}
	args := []string{"windowfocus", id}
	switch v.Kind {
	case "pointer":
		args = append(args, "mousemove", "--window", id, strconv.Itoa(v.X), strconv.Itoa(v.Y))
		if v.Button > 0 {
			action := "mouseup"
			if v.Down {
				action = "mousedown"
			}
			if v.Button >= 4 {
				action = "click"
			}
			args = append(args, action, strconv.Itoa(v.Button))
		}
	case "key":
		args = append(args, "key", "--clearmodifiers", v.Key)
	case "text":
		args = append(args, "type", "--clearmodifiers", "--delay", "0", "--", v.Text)
	}
	c, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	return exec.CommandContext(c, "/usr/bin/xdotool", args...).Run()
}
func servePrivateFrames(ctx context.Context, w *mygo.Window, input io.Reader, output io.Writer, nonce string, doc Document) error {
	startup, stop := context.WithTimeout(ctx, 25*time.Second)
	defer stop()
	loaded := make(chan error, 1)
	deliver := func(err error) {
		select {
		case loaded <- err:
		default:
		}
	}
	offLoad := w.Page().OnDidFinishLoad(func() { deliver(nil) })
	defer offLoad()
	offFail := w.Page().OnDidFailLoad(func(e *mygo.LoadError) { deliver(e) })
	defer offFail()
	if err := w.Page().LoadURL("about:blank"); err != nil {
		return err
	}
	select {
	case err := <-loaded:
		if err != nil {
			return err
		}
	case <-startup.Done():
		return startup.Err()
	}
	shell, err := htmlpreview.ProcessDocumentShell(doc.HTML, doc.Width, doc.Appearance)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(shell)
	if err != nil {
		return err
	}
	w.SetContentSize(doc.Width, interactiveHeight)
	w.Show() // Visible only on the private Xvfb display.
	if _, err = w.Page().EvalContext(startup, "document.open();document.write("+string(encoded)+");document.close();true"); err != nil {
		return err
	}
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
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
		if err = json.Unmarshal(raw, &state); err != nil {
			return err
		}
		if state.Error != "" {
			return errors.New("private preview document failed to initialize")
		}
		if state.Ready {
			break
		}
		select {
		case <-startup.Done():
			return startup.Err()
		case <-tick.C:
		}
	}
	if _, err := w.Page().EvalContext(startup, "new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)))"); err != nil {
		return err
	}
	id, err := privateInputWindow(startup)
	if err != nil {
		return err
	}
	// Do not advertise loaded until a real bounded frame has been captured.
	first, err := w.CapturePage()
	if err != nil {
		return err
	}
	if err = validateFrame(first, doc.Width); err != nil {
		return err
	}
	if err = writePacket(output, packet{Version: protocolVersion, Kind: "loaded", Nonce: nonce}); err != nil {
		return err
	}
	if err = sendFrame(output, nonce, first, doc.Width); err != nil {
		return err
	}
	for {
		p, err := readPacket(input)
		if err != nil {
			return err
		}
		if p.Version != protocolVersion || p.Nonce != nonce {
			return errors.New("invalid private preview command")
		}
		switch p.Kind {
		case "poll":
			if p.Input != nil {
				if err = applyPrivateInput(ctx, id, *p.Input, doc.Width); err != nil {
					return err
				}
			}
		case "theme":
			script, e := htmlpreview.ProcessThemeTokensScript(p.Appearance, p.Variables)
			if e != nil {
				return e
			}
			if _, err = w.Page().EvalContext(ctx, script); err != nil {
				return err
			}
		default:
			return errors.New("unexpected private preview command")
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		frame, err := w.CapturePage()
		if err != nil {
			return err
		}
		if err = sendFrame(output, nonce, frame, doc.Width); err != nil {
			return err
		}
	}
}
