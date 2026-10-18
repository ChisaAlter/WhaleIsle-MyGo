package htmlpreview

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

// RendererService binds status, the fixed runtime probe, and capture to one
// renderer. In particular a helper probe never grants in-process readiness.
// The supplied renderer must enforce its own isolation on every invocation.
type RendererService struct {
	render    Renderer
	mu        sync.RWMutex
	operation sync.Mutex
	status    ReadinessStatus
}

func NewRendererService(render Renderer) *RendererService {
	return &RendererService{render: render, status: ReadinessStatus{Reason: UnavailableReason}}
}

func (s *RendererService) Readiness() ReadinessStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *RendererService) setStatus(status ReadinessStatus) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
}

// Probe invokes exactly the renderer used by this service's POST route, with
// synthetic source only. It does not change the user's enabled preference.
func (s *RendererService) Probe(ctx context.Context) (status ReadinessStatus) {
	s.operation.Lock()
	defer s.operation.Unlock()
	status = ReadinessStatus{Reason: "HTML preview runtime verification is in progress."}
	s.setStatus(status)
	defer func() {
		if recover() != nil {
			status = ReadinessStatus{Reason: "HTML preview helper runtime verification failed."}
		}
		s.setStatus(status)
	}()
	if s.render == nil {
		return ReadinessStatus{Reason: UnavailableReason}
	}
	ctx, cancel := context.WithTimeout(ctx, CaptureTimeout)
	defer cancel()
	result, err := s.render(ctx, probeHTML, 240, "dark")
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return ReadinessStatus{Reason: "HTML preview helper runtime verification failed: " + err.Error()}
	}
	if result.Width != 240 || result.ContentHeight < 1 || result.CapturedHeight < 1 || result.PNG == "" {
		return ReadinessStatus{Reason: "HTML preview helper returned an invalid capture."}
	}
	verified := false
	for _, message := range result.ConsoleMessages {
		if strings.HasPrefix(message.Text, "preview-probe-failed:") {
			return ReadinessStatus{Reason: message.Text}
		}
		if message.Text == "preview-probe-passed:opaque/no-native/no-storage/no-rtc" {
			verified = true
		}
	}
	if !verified {
		return ReadinessStatus{Reason: "HTML preview helper did not confirm the opaque sandbox boundaries."}
	}
	return ReadinessStatus{Available: true}
}

func (s *RendererService) capture(ctx context.Context, html string, width int, appearance string) (CaptureResult, error) {
	s.operation.Lock()
	defer s.operation.Unlock()
	if err := ctx.Err(); err != nil {
		return CaptureResult{}, err
	}
	if status := s.Readiness(); !status.Available {
		return CaptureResult{}, errors.New(status.Reason)
	}
	result, err := s.render(ctx, html, width, appearance)
	// An operation failure invalidates this renderer alone; subsequent captures
	// require another fixed probe. Never publish ready from arbitrary HTML.
	if err != nil {
		s.setStatus(ReadinessStatus{Reason: "HTML preview helper capture failed: " + err.Error()})
	}
	return result, err
}

func (s *RendererService) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() != nil {
			return
		}
		if r.URL.Path == "/desktop/html-preview/status" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_ = json.NewEncoder(w).Encode(s.Readiness())
			return
		}
		serveWithRenderer(w, r, s.capture)
	})
}
