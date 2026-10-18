package settings

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh"
)

func TestBusyEnterReadsResolvedExactHostOnly(t *testing.T) {
	a, b := &dsh.Client{}, &dsh.Client{}
	s := New()
	s.Attach(a, nil)
	s.describedClient = a
	s.namespaces = []Namespace{{NS: "conversation", Value: json.RawMessage(`{"busyEnter":"steer"}`)}, {NS: "ui-conversation", Value: json.RawMessage(`{"busyEnter":"queue"}`)}}
	s.selectedNS = "ui-conversation"
	s.SetDraft("busyEnter", "steer")
	if got := s.BusyEnterFor(a); got != "queue" {
		t.Fatalf("unsaved/alias value consumed: %s", got)
	}
	s.namespaces[1].Value = json.RawMessage(`{"busyEnter":"steer"}`)
	if got := s.BusyEnterFor(a); got != "steer" {
		t.Fatal(got)
	}
	if got := s.BusyEnterFor(b); got != "queue" {
		t.Fatal("different Host adopted settings")
	}
	s.Attach(b, nil) // same origin must not reuse the previous client's describe
	if got := s.BusyEnterFor(b); got != "queue" {
		t.Fatal("same-origin replacement adopted old describe")
	}
	s.describedClient = b
	for _, raw := range []string{`{}`, `{"busyEnter":"invalid"}`, `{"busyEnter":true}`, `{`} {
		s.namespaces[1].Value = json.RawMessage(raw)
		if got := s.BusyEnterFor(b); got != "queue" {
			t.Fatalf("invalid default %s: %s", raw, got)
		}
	}
	s.Detach()
	if got := s.BusyEnterFor(b); got != "queue" {
		t.Fatal("detached Host settings leaked")
	}
}
