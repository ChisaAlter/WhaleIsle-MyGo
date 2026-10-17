package mygo

import "errors"

// EnableIsolatedOpenUIFrames selects the Windows-only, source-bounded frame
// policy for a trusted OpenUI shell before any content is loaded. The caller
// must first verify its contained helper and the parent document's OpenUI mode.
// It admits one opaque srcdoc viewer and that viewer's immediate srcdoc children.
// It is not an isolation attestation and must never be bound to web content.
// Network, permissions, native bindings and all other preview restrictions stay
// unchanged. Unsupported backends and an already-used preview fail closed.
func (p *Page) EnableIsolatedOpenUIFrames() error {
	return onMainValue(func() error {
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil {
			return errDestroyed
		}
		if !p.w.disableBindings {
			return errors.New("mygo: OpenUI preview requires disabled bindings")
		}
		n, ok := p.w.native.(interface{ EnableIsolatedOpenUIFrames() error })
		if !ok {
			return errors.New("mygo: isolated OpenUI frame tracking is unavailable")
		}
		return n.EnableIsolatedOpenUIFrames()
	})
}

// IsolatedOpenUIFrameSupport reports whether the actual viewer frame has been
// observed, and whether its immediate HtmlView children can be tracked. It is
// read by the contained helper to report unsupported nested frames explicitly;
// it grants no permissions and is not available to JavaScript.
func (p *Page) IsolatedOpenUIFrameSupport() (known, htmlView bool, err error) {
	type result struct {
		known, htmlView bool
		err             error
	}
	r := onMainValue(func() result {
		if p == nil || p.w == nil || p.w.native == nil || !p.w.disableBindings {
			return result{err: errDestroyed}
		}
		n, ok := p.w.native.(interface{ IsolatedOpenUIFrameSupport() (bool, bool, error) })
		if !ok {
			return result{err: errors.New("mygo: isolated OpenUI frame tracking is unavailable")}
		}
		known, htmlView, err := n.IsolatedOpenUIFrameSupport()
		return result{known, htmlView, err}
	})
	return r.known, r.htmlView, r.err
}
