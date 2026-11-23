package state

import (
	"errors"
	"fmt"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
)

// Fault is the classified, render-ready form of a failure. It never keeps
// the raw error text: the boundary that caught the error logs it, and the
// view renders Sentence. Code is the Host's stable code or the HTTP status
// when one was observed, so the sentence stays stable across locales.
type Fault struct {
	Cause     errcopy.Cause
	Code      string
	Transient bool
}

// Sentence renders the localized failure sentence for locale; "" follows
// the active catalog.
func (f *Fault) Sentence(locale string) string {
	if f == nil {
		return ""
	}
	return errcopy.CauseSentenceIn(locale, f.Cause, f.Code)
}

// SentenceText renders the sentence in the active catalog.
func (f *Fault) SentenceText() string {
	return f.Sentence("")
}

// FaultOf classifies err into a Fault, or returns nil for nil. Typed dsh
// carriers contribute their stable code, HTTP status and transience; the
// raw message stays with the caller's log line.
func FaultOf(err error) *Fault {
	if err == nil {
		return nil
	}
	f := &Fault{Cause: errcopy.Classify(err)}
	var transport *dsh.TransportError
	if errors.As(err, &transport) {
		f.Transient = transport.Temporary()
	}
	var httpErr *dsh.HTTPError
	if errors.As(err, &httpErr) {
		f.Code = fmt.Sprintf("HTTP_%d", httpErr.Status)
		f.Transient = f.Transient || httpErr.Temporary()
	}
	var rpcErr *dsh.Error
	if errors.As(err, &rpcErr) && rpcErr.Code != "" {
		f.Code = rpcErr.Code
	}
	if errors.Is(err, dsh.ErrEndpointNotReady) {
		f.Transient = true
	}
	if f.Cause == errcopy.CauseTimeout {
		f.Transient = true
	}
	return f
}
