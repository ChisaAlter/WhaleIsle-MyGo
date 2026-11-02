package session

import (
	"context"
	"errors"
	"sync"
	"whaleisle/internal/dsh"
	"whaleisle/internal/jobflow"
)

type nativeJobsState struct {
	mu         sync.Mutex
	controller *jobflow.Controller
	client     *dsh.Client
	mux        *dsh.Mux
	sessionID  string
}

// NativeJobs owns one panel bound to the selected session and exact Host/mux.
// All invalidations use the same session-generation fence as transcript work.
func (s *Store) NativeJobs() *jobflow.Controller {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := &s.jobs
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.controller == nil {
		j.controller = jobflow.New(s.fire)
	}
	if j.client != s.client || j.mux != s.mux || j.sessionID != s.selectedID {
		var transport jobflow.Transport
		if s.client != nil && s.mux != nil && s.selectedID != "" {
			transport = &nativeJobTransport{client: s.client, mux: s.mux}
		}
		j.controller.Bind(transport, s.selectedID)
		j.client = s.client
		j.mux = s.mux
		j.sessionID = s.selectedID
	}
	return j.controller
}
func (s *Store) invalidateJobsLocked() {
	s.invalidateViewLocked()
	s.jobs.mu.Lock()
	if s.jobs.controller != nil {
		s.jobs.controller.Bind(nil, "")
	}
	s.jobs.client = nil
	s.jobs.mux = nil
	s.jobs.sessionID = ""
	s.jobs.mu.Unlock()
}

// UnmountNativeJobs releases only the logical jobs streams when its header is
// absent, including top-level pages that replace the conversation frame.
func (s *Store) UnmountNativeJobs() {
	s.jobs.mu.Lock()
	controller := s.jobs.controller
	s.jobs.mu.Unlock()
	if controller != nil {
		controller.Unmount()
	}
}

type nativeJobTransport struct {
	client *dsh.Client
	mux    *dsh.Mux
}

func (t *nativeJobTransport) Open(ctx context.Context, method string, request map[string]any) (jobflow.Stream, error) {
	if err := t.mux.Dial(ctx); err != nil {
		return nil, err
	}
	return t.mux.Open(ctx, method, map[string]any{"args": map[string]any{"request": request}})
}
func (t *nativeJobTransport) Kill(ctx context.Context, request map[string]any) (string, error) {
	var out struct {
		Outcome string `json:"outcome"`
	}
	err := t.client.Call(ctx, "job/kill", map[string]any{"request": request}, &out)
	return out.Outcome, err
}
func (*nativeJobTransport) Retryable(err error) bool {
	return errors.Is(err, dsh.ErrCarrierLost) || errors.Is(err, dsh.ErrStreamEnd)
}
