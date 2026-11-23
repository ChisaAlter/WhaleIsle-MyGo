package state

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"
)

func TestFaultOfNil(t *testing.T) {
	if FaultOf(nil) != nil {
		t.Fatal("nil error must produce nil fault")
	}
}

func TestFaultOfTransportError(t *testing.T) {
	err := &dsh.TransportError{Op: "session/list", Err: fmt.Errorf("read tcp: %w", syscall.ECONNRESET)}
	f := FaultOf(err)
	if f == nil || f.Cause != errcopy.CauseNetwork || !f.Transient {
		t.Fatalf("fault %#v", f)
	}
	s := f.SentenceText()
	if s == "" || strings.Contains(s, "read tcp") {
		t.Fatalf("sentence leaked raw text or is empty: %q", s)
	}
}

func TestFaultOfHTTPError(t *testing.T) {
	err := &dsh.HTTPError{Op: "chat/send", Status: 429, Body: "slow down"}
	f := FaultOf(err)
	if f == nil || f.Code != "HTTP_429" || !f.Transient {
		t.Fatalf("fault %#v", f)
	}
	if s := f.Sentence("en"); !strings.Contains(s, "429") {
		t.Fatalf("rejected sentence must quote the status: %q", s)
	}
	if s := f.Sentence("zh-CN"); !strings.Contains(s, "限流") {
		t.Fatalf("zh sentence %q", s)
	}
	// A 404 is a rejection, not a transient outage.
	err404 := &dsh.HTTPError{Op: "x", Status: 404}
	if FaultOf(err404).Transient {
		t.Fatal("404 must not be transient")
	}
}

func TestFaultOfRPCErrorCodeAndEndpointNotReady(t *testing.T) {
	err := &dsh.Error{Code: "MISSING_CREDENTIAL", Message: "no api key configured"}
	f := FaultOf(err)
	if f == nil || f.Code != "MISSING_CREDENTIAL" || f.Cause != errcopy.CauseMissingCredential {
		t.Fatalf("fault %#v", f)
	}
	if s := f.Sentence("en"); !strings.Contains(s, "API key") {
		t.Fatalf("credential sentence %q", s)
	}
	notReady := &dsh.Error{Code: "gateway/service-unavailable", Message: "service unavailable"}
	if !errors.Is(notReady, dsh.ErrEndpointNotReady) {
		t.Fatal("service-unavailable must match ErrEndpointNotReady")
	}
	nf := FaultOf(notReady)
	if nf == nil || !nf.Transient || nf.Cause != errcopy.CauseTimeout {
		t.Fatalf("endpoint-not-ready fault %#v", nf)
	}
}

func TestFaultNilSafety(t *testing.T) {
	var f *Fault
	if f.Sentence("en") != "" || f.SentenceText() != "" {
		t.Fatal("nil fault must render empty")
	}
}

func TestFaultLocale(t *testing.T) {
	f := &Fault{Cause: errcopy.CauseNetwork}
	en, zh := f.Sentence("en"), f.Sentence("zh-CN")
	if en == "" || zh == "" || en == zh {
		t.Fatalf("locale switch did not change the sentence: %q / %q", en, zh)
	}
	if i18n.Normalize("en") != "en" {
		t.Fatal("catalog locale assumption broken")
	}
}
