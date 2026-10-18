package mygo

import "errors"

// BrowserProcessID returns a native browser PID for an already initialized
// WebView2. A PID is an observation, never an isolation attestation. The caller
// must independently validate its token and lifetime-bound job membership.
func (p *Page) BrowserProcessID() (uint32, error) {
	type result struct {
		pid uint32
		err error
	}
	r := onMainValue(func() result {
		if p == nil || p.w == nil || p.w.native == nil {
			return result{err: errDestroyed}
		}
		n, ok := p.w.native.(interface{ BrowserProcessID() (uint32, error) })
		if !ok {
			return result{err: errors.New("mygo: browser process identity is unavailable")}
		}
		pid, err := n.BrowserProcessID()
		return result{pid, err}
	})
	if r.pid == 0 && r.err == nil {
		return 0, errDestroyed
	}
	return r.pid, r.err
}
