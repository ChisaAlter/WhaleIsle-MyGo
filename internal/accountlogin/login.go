// Package accountlogin owns only transient desktop login presentation state.
package accountlogin

import (
	"strings"
	"sync"
	"whaleisle/internal/dsh/api"
)

type State struct {
	Busy       bool
	Visible    bool
	URL, Error string
}
type Flow struct {
	mu      sync.Mutex
	state   State
	attempt string
	opened  map[string]bool
}

func Active(phase string) bool {
	switch phase {
	case "initializing", "waiting-browser", "exchanging", "committing":
		return true
	}
	return false
}

// Begin is called only by an explicit login click, before dispatching the RPC.
func (f *Flow) Begin() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state.Busy || f.attempt != "" {
		return false
	}
	f.state = State{Busy: true, Visible: true}
	f.attempt = ""
	return true
}
func (f *Flow) Finish() { f.mu.Lock(); f.attempt = ""; f.state = State{}; f.mu.Unlock() }

func (f *Flow) BeginCancel() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state.Busy {
		return false
	}
	f.state.Busy = true
	return true
}
func (f *Flow) CancelFailed(message string) {
	f.mu.Lock()
	f.state.Busy = false
	f.state.Error = message
	f.mu.Unlock()
}

func (f *Flow) Snapshot() State { f.mu.Lock(); defer f.mu.Unlock(); return f.state }
func (f *Flow) Started(v api.AccountView, err error, open func(string) error) {
	f.mu.Lock()
	f.state.Busy = false
	if err != nil {
		f.state.Error = err.Error()
		f.mu.Unlock()
		return
	}
	f.attempt = v.Attempt.Id
	f.mu.Unlock()
	f.Observe(v, open)
}

// Observe never opens a login not initiated in this process. Mark before opening
// so reconnects, duplicate frames and browser failures cannot trigger a retry.
func (f *Flow) Observe(v api.AccountView, open func(string) error) {
	f.mu.Lock()
	if v.Status == "credential-stored" {
		f.attempt = ""
		f.state = State{}
		f.mu.Unlock()
		return
	}
	if f.attempt == "" || f.attempt != v.Attempt.Id {
		f.mu.Unlock()
		return
	}
	if v.Attempt.AuthorizeUrl != nil && strings.TrimSpace(*v.Attempt.AuthorizeUrl) != "" {
		f.state.URL = strings.TrimSpace(*v.Attempt.AuthorizeUrl)
	}
	if !Active(v.Attempt.Phase) {
		code := ""
		if v.Attempt.ErrorCode != nil {
			code = *v.Attempt.ErrorCode
		}
		f.state.Error = FailureMessage(v.Attempt.Phase, code)
		f.state.URL = ""
		f.attempt = ""
		f.mu.Unlock()
		return
	}
	id, url := f.attempt, f.state.URL
	if v.Attempt.Phase != "waiting-browser" || url == "" || f.opened[id] || open == nil {
		f.mu.Unlock()
		return
	}
	if f.opened == nil {
		f.opened = map[string]bool{}
	}
	f.opened[id] = true
	f.mu.Unlock()
	if err := open(url); err != nil {
		f.mu.Lock()
		if f.attempt == id {
			f.state.Error = err.Error()
		}
		f.mu.Unlock()
	}
}
