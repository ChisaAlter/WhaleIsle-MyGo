package htmlpreview

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// ReadinessStatus distinguishes missing/unverified capture support from the
// independent user preference (Visual replies remains off by default).
type ReadinessStatus struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

var readiness = struct {
	sync.RWMutex
	status ReadinessStatus
}{status: ReadinessStatus{Reason: UnavailableReason}}
var probeMu sync.Mutex

func Readiness() ReadinessStatus {
	readiness.RLock()
	defer readiness.RUnlock()
	return readiness.status
}
func Available() bool { return Readiness().Available }
func setReadiness(status ReadinessStatus) {
	readiness.Lock()
	readiness.status = status
	readiness.Unlock()
}

// Probe is an explicit, bounded native runtime check using ONLY the fixed
// synthetic fixture below. It accepts no HTML, path, URL, profile or callback.
// It must run while the desktop GUI loop is active, off that loop's thread.
// Ordinary window/Reader existence and Windows compilation cannot mark ready.
// A failed check leaves all preview capture calls unavailable. The user's
// visual-reply generation setting is never changed by this check.
func Probe(ctx context.Context) (status ReadinessStatus) {
	probeMu.Lock()
	defer probeMu.Unlock()
	setReadiness(ReadinessStatus{Reason: "HTML preview runtime verification is in progress."})
	defer func() {
		if recover() != nil {
			status = ReadinessStatus{Reason: "HTML preview runtime verification failed while creating the isolated context."}
			setReadiness(status)
		}
	}()
	result, err := captureNative(ctx, request{HTML: probeHTML, Width: 240, Appearance: "dark"})
	if err != nil {
		status = ReadinessStatus{Reason: "HTML preview runtime verification failed: " + err.Error()}
		setReadiness(status)
		return
	}
	if result.Width != 240 || result.ContentHeight < 1 || result.CapturedHeight < 1 || result.PNG == "" {
		status = ReadinessStatus{Reason: "HTML preview runtime verification returned an invalid native capture."}
		setReadiness(status)
		return
	}
	verified := false
	for _, message := range result.ConsoleMessages {
		if strings.HasPrefix(message.Text, "preview-probe-failed:") {
			status = ReadinessStatus{Reason: message.Text}
			setReadiness(status)
			return
		}
		if message.Text == "preview-probe-passed:opaque/no-native/no-storage/no-rtc" {
			verified = true
		}
	}
	if !verified {
		status = ReadinessStatus{Reason: "HTML preview runtime did not confirm the opaque sandbox boundaries."}
		setReadiness(status)
		return
	}
	status = ReadinessStatus{Available: true}
	setReadiness(status)
	return
}

// No network destinations or real file paths are in this fixture. Probe does
// not mistake DNS failures for a proven network policy. Native creation has
// already checked effective private UDF and every deny-policy registration.
// WebRTC has transports outside ordinary URL requests; if those constructors
// are exposed we cannot attest to complete network isolation and stay closed.
const probeHTML = `<!doctype html><body><p>Isolated preview fixture</p><script>(()=>{
 const failures=[];
 try{void parent.document;failures.push('parent document accessible')}catch{}
 try{void localStorage.length;failures.push('persistent storage accessible')}catch{}
 if(typeof window.mygo!=='undefined')failures.push('native bindings exposed');
 if(typeof RTCPeerConnection!=='undefined'||typeof webkitRTCPeerConnection!=='undefined'||typeof RTCIceGatherer!=='undefined')failures.push('WebRTC transport isolation not verified');
 console.log(failures.length?'preview-probe-failed:'+failures.join('; '):'preview-probe-passed:opaque/no-native/no-storage/no-rtc');
})();</script></body>`

// RequireAvailable is the shared boundary for a future historical Reader.
// The Reader must still create its own IsolatedPreview window, use the strict
// sandbox shell, and own its close lifecycle. It must never borrow a normal
// browsing context or treat DisableBindings by itself as isolation.
func RequireAvailable() error {
	if s := Readiness(); !s.Available {
		return errors.New(s.Reason)
	}
	return nil
}

// SandboxColors contains trusted shell theme values, never filesystem paths.
// It is shared with the saved-HTML Reader so the Reader need not duplicate the
// opaque-frame/CSP wrapper or relax it to use an ordinary browsing context.
type SandboxColors = previewPalette

// SandboxDocument builds source only. It grants no readiness and opens no
// window. A Reader must RequireAvailable, create PageOptions.IsolatedPreview
// with DisableBindings, and show that independent window only after LoadBlank
// and this document finish loading; it must destroy it on close/cancellation.
func SandboxDocument(preparedHTML string, width int, appearance string, colors SandboxColors) (string, error) {
	if strings.TrimSpace(preparedHTML) == "" || len(preparedHTML) > MaxPreparedHTMLBytes {
		return "", errors.New("Prepared HTML must contain 1 to 25 MiB of UTF-8 content.")
	}
	if width < MinWidth || width > MaxWidth {
		return "", errors.New("Preview width must be an integer from 240 to 1600.")
	}
	if appearance != "light" && appearance != "dark" {
		return "", errors.New("Preview appearance must be light or dark.")
	}
	return buildShell(request{HTML: preparedHTML, Width: width, Appearance: appearance}, colors), nil
}
