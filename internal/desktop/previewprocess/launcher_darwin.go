//go:build darwin && (amd64 || arm64)

package previewprocess

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Legacy WebView is a deprecated single-process compatibility renderer, not a
// WKWebView equivalent. It runs only inside its independently signed sandbox.
func openPlatform(ctx context.Context, opt Options, doc Document) (*Session, error) {
	if doc.Inline && doc.capture {
		return nil, errors.New("macOS capture requires a capture-only helper session")
	}
	setup, stop := context.WithTimeout(ctx, 20*time.Second)
	defer stop()
	identity, err := verifyMacBundle(setup, opt.HelperPath)
	if err != nil {
		return nil, err
	}
	shell, err := documentShell(doc)
	if err != nil {
		return nil, err
	}
	wireDoc := doc
	wireDoc.HTML = shell
	wireDoc.openUIShell = doc.OpenUI != nil
	wireDoc.OpenUI = nil
	if err = validateDocument(wireDoc); err != nil {
		return nil, err
	}
	var random [32]byte
	if _, err = rand.Read(random[:]); err != nil {
		return nil, err
	}
	nonce := hex.EncodeToString(random[:])
	lifetime, cancel := context.WithTimeout(ctx, 30*time.Minute)
	cmd := exec.CommandContext(lifetime, opt.HelperPath, "--legacy-sandbox-helper-v1", nonce)
	// No DYLD, proxy, credential, shared profile, HOME or application overrides.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=en_US.UTF-8"}
	cmd.Stderr = os.Stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		cancel()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		cancel()
		return nil, err
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	session := &Session{done: make(chan struct{})}
	session.initOpenUIEvents(doc)
	session.closeFn = func() error {
		_ = input.Close()
		cancel()
		_ = cmd.Process.Kill()
		<-exited
		_ = output.Close()
		return nil
	}
	var writeMu sync.Mutex
	session.sendTheme = func(appearance string, vars map[string]string) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writePacket(input, packet{Version: protocolVersion, Kind: "theme", Nonce: nonce, Appearance: appearance, Variables: vars})
	}
	var firstFrame []byte
	ready := make(chan error, 1)
	go func() {
		p, e := readPacket(output)
		if e == nil {
			e = expect(p, "ready", nonce)
		}
		if e == nil {
			e = requireOpenUIBridge(doc, p)
			session.enableOpenUIUpdates(p.OpenUIUpdate)
			session.enableOpenUIAPI(p.OpenUIAPI)
		}
		if e == nil && (p.HelperPID != uint32(cmd.Process.Pid) || p.BrowserPID != p.HelperPID) {
			e = errors.New("legacy renderer is not the launched single-process helper")
		}
		if e == nil {
			e = verifyMacProcess(cmd.Process.Pid, identity)
		}
		// Only after the kernel-identified child matches the checked signed image
		// does this pipe receive any untrusted document bytes.
		if e == nil {
			e = sendDocument(input, nonce, wireDoc)
		}
		if e == nil {
			p, e = readPacket(output)
			if e == nil {
				e = expect(p, "loaded", nonce)
			}
		}
		if e == nil && p.BrowserPID != uint32(cmd.Process.Pid) {
			e = errors.New("legacy renderer identity changed")
		}
		if e == nil {
			e = verifyMacProcess(cmd.Process.Pid, identity)
		}
		if e == nil && doc.capture {
			result, captureErr := receiveCapture(output, nonce, doc.Width)
			e = captureErr
			if e == nil {
				session.capture = &result
			}
		}
		if e == nil && doc.Inline {
			firstFrame, e = receiveFrame(output, nonce, doc.Width)
		}
		ready <- e
	}()
	select {
	case err = <-ready:
	case <-setup.Done():
		err = setup.Err()
	case <-exited:
		err = errors.New("signed legacy preview helper exited before initialization")
	}
	if err != nil {
		session.Close()
		return nil, err
	}
	if doc.Inline {
		if err = startMacPixelSeat(lifetime, session, input, output, nonce, doc, firstFrame); err != nil {
			session.Close()
			return nil, err
		}
	}
	if doc.OpenUI != nil && !doc.Inline {
		startOpenUIAPIPump(lifetime, session, func(raw []byte) error {
			writeMu.Lock()
			defer writeMu.Unlock()
			return sendOpenUIAPIResponse(input, nonce, raw)
		})
		startOpenUIUpdatePump(lifetime, session, func(raw []byte) error {
			writeMu.Lock()
			defer writeMu.Unlock()
			return sendOpenUIUpdate(input, nonce, raw)
		})
		go func() {
			defer session.Close()
			for {
				p, e := readPacket(output)
				if e != nil || session.receiveOpenUIEvent(output, p, nonce) != nil {
					return
				}
			}
		}()
	}
	go func() {
		select {
		case <-exited:
		case <-lifetime.Done():
		}
		_ = session.Close()
	}()
	return session, nil
}
