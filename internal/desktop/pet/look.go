package pet

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"image/jpeg"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/screenglance"
	"whaleisle/internal/store/storelocale"
)

type LookStatus struct {
	Busy    bool
	Message string
	ID      uint64
	Pinned  bool
	Expires int64
}
type lookEntry struct {
	status      LookStatus
	serial      uint64
	cancel      context.CancelFunc
	lastCapture time.Time
}

func (e *lookEntry) begin(now time.Time) (context.Context, uint64) {
	if e.status.Busy {
		return nil, 0
	}
	if !e.lastCapture.IsZero() && now.Sub(e.lastCapture) < 4*time.Second {
		e.serial++
		e.status = LookStatus{ID: e.serial, Message: storelocale.T(storelocale.KeyDesktopPetPleaseWaitBeforeAnotherGlance), Expires: now.Add(4 * time.Second).UnixMilli()}
		return nil, 0
	}
	e.serial++
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.status = LookStatus{Busy: true, ID: e.serial, Message: storelocale.T(storelocale.KeyDesktopPetWaitingForScreenshotApproval)}
	return ctx, e.serial
}
func (e *lookEntry) finish(serial uint64, message string) bool {
	if e.serial != serial || !e.status.Busy {
		return false
	}
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.status = LookStatus{ID: serial, Message: message, Expires: time.Now().Add(9 * time.Second).UnixMilli()}
	return true
}
func (e *lookEntry) close() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.serial++
	e.status = LookStatus{}
}

// LookAttempt is tied to one pet window, with no capability exposed to its page.
// Methods that inspect Window run on the UI thread.
type LookAttempt struct {
	Context context.Context
	Window  *mygo.Window
	pet     *Companion
	serial  uint64
}

func (p *Companion) StartLook() *LookAttempt {
	if p.win == nil || p.win.IsDestroyed() {
		return nil
	}
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	ctx, serial := p.look.begin(time.Now())
	if ctx == nil {
		return nil
	}
	return &LookAttempt{Context: ctx, Window: p.win, pet: p, serial: serial}
}
func (a *LookAttempt) Current() bool {
	if a == nil || a.Context.Err() != nil || a.pet.win != a.Window || a.Window.IsDestroyed() {
		return false
	}
	a.pet.assetMu.RLock()
	defer a.pet.assetMu.RUnlock()
	return a.pet.look.serial == a.serial && a.pet.look.status.Busy
}
func (a *LookAttempt) CaptureStarted() bool {
	if !a.Current() {
		return false
	}
	a.pet.assetMu.Lock()
	defer a.pet.assetMu.Unlock()
	a.pet.look.lastCapture = time.Now()
	a.pet.look.status.Message = storelocale.T(storelocale.KeyDesktopPetLooking)
	return true
}
func (a *LookAttempt) Finish(message string) bool {
	if a == nil {
		return false
	}
	a.pet.assetMu.Lock()
	defer a.pet.assetMu.Unlock()
	return a.pet.look.finish(a.serial, message)
}

// Reply pins only a successful result. Cancel/error feedback expires rather
// than permanently blocking the notification queue.
func (a *LookAttempt) Reply(message string) bool {
	if a == nil {
		return false
	}
	a.pet.assetMu.Lock()
	defer a.pet.assetMu.Unlock()
	if !a.pet.look.finish(a.serial, message) {
		return false
	}
	a.pet.look.status.Pinned = message != ""
	a.pet.look.status.Expires = 0
	return true
}

func renderLookPage(page []byte, status LookStatus) []byte {
	return []byte(strings.ReplaceAll(string(page), "{{LOOK_STATUS}}", html.EscapeString(status.Message)))
}

// SendGlance only calls the existing authenticated Whale pet/look route. It
// does not choose credentials, substitute a model, retry, or log the image.
func SendGlance(ctx context.Context, origin, cookie, provider, model string, frame []byte) (string, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.RawPath != "" || u.Fragment != "" {
		return "", fmt.Errorf("pet look: invalid Host origin")
	}
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(model) == "" || len(frame) == 0 || len(frame) > 1<<20 {
		return "", fmt.Errorf("pet look: missing route or invalid frame")
	}
	shape, err := jpeg.DecodeConfig(bytes.NewReader(frame))
	if err != nil || shape.Width < 1 || shape.Height < 1 || shape.Width > screenglance.MaxWidth || shape.Height > screenglance.MaxHeight {
		return "", fmt.Errorf("pet look: invalid JPEG dimensions")
	}
	client := &pluginrpc.Client{BaseURL: origin, Cookie: cookie, HTTPClient: &http.Client{
		Timeout:       60 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
	raw, err := client.WhalePetLook(ctx, provider, model, base64.StdEncoding.EncodeToString(frame))
	if err != nil {
		return "", err
	}
	var result struct {
		OK    bool   `json:"ok"`
		Reply string `json:"reply"`
	}
	if json.Unmarshal(raw, &result) != nil || !result.OK || strings.TrimSpace(result.Reply) == "" {
		return "", fmt.Errorf("pet look: no successful reply")
	}
	reply := []rune(strings.TrimSpace(result.Reply))
	if len(reply) > 240 {
		reply = reply[:240]
	}
	return string(reply), nil
}
