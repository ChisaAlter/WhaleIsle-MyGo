package openui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestHTMLScriptValidationAndHardening(t *testing.T) {
	id := Identity("host", "session", "message", 0)
	api := NewResourceAPI()

	// Empty src, unknown query params, and oversized query budgets are rejected.
	badRequests := []Event{
		{InstanceID: id, RequestID: "empty", Path: "/api/html-script", Query: map[string]string{}},
		{InstanceID: id, RequestID: "blank", Path: "/api/html-script", Query: map[string]string{"src": "   "}},
		{InstanceID: id, RequestID: "extra", Path: "/api/html-script", Query: map[string]string{"src": "https://cdn.jsdelivr.net/npm/a.js", "headers": "x"}},
		{InstanceID: id, RequestID: "oversize-src", Path: "/api/html-script", Query: map[string]string{"src": "https://cdn.jsdelivr.net/" + strings.Repeat("a", MaxEventBytes)}},
		{InstanceID: id, RequestID: "oversize-sri", Path: "/api/html-script", Query: map[string]string{"src": "https://cdn.jsdelivr.net/npm/a.js", "integrity": strings.Repeat("a", MaxEventBytes+1)}},
	}
	for _, ev := range badRequests {
		if err := validateAPIRequest(ev); err == nil {
			t.Fatalf("validateAPIRequest unexpectedly accepted %q", ev.RequestID)
		}
		if res := api.Handle(context.Background(), ev); res.Status != 400 {
			t.Fatalf("Handle(%q) status = %d, want 400", ev.RequestID, res.Status)
		}
	}

	// Non-HTTPS URLs and invalid SRI digests fail closed at status 400.
	for _, ev := range []Event{
		{InstanceID: id, RequestID: "http", Path: "/api/html-script", Query: map[string]string{"src": "http://cdn.jsdelivr.net/npm/a.js"}},
		{InstanceID: id, RequestID: "bad-sri", Path: "/api/html-script", Query: map[string]string{"src": "https://cdn.jsdelivr.net/npm/a.js", "integrity": "sha1-deadbeef"}},
	} {
		if res := api.Handle(context.Background(), ev); res.Status != 400 {
			t.Fatalf("Handle(%q) status = %d, want 400 (body=%s)", ev.RequestID, res.Status, string(res.Body))
		}
	}

	// Private/loopback/internal URLs fail closed at status 403 before dialing.
	for _, raw := range []string{
		"https://127.0.0.1/evil.js",
		"https://169.254.169.254/latest/meta-data",
		"https://10.0.0.1/internal.js",
		"https://localhost/local.js",
	} {
		res := api.Handle(context.Background(), Event{
			InstanceID: id,
			RequestID:  "priv",
			Path:       "/api/html-script",
			Query:      map[string]string{"src": raw},
		})
		if res.Status != 403 {
			t.Fatalf("private script %q status = %d, want 403", raw, res.Status)
		}
	}

	// Oversized JSON events (> MaxEventBytes) remain rejected by ValidateEvent.
	oversized, _ := json.Marshal(Event{
		Channel:    "whale-openui",
		Version:    1,
		InstanceID: id,
		Type:       "api-request",
		RequestID:  "big",
		Path:       "/api/html-script",
		Query:      map[string]string{"src": "https://cdn.jsdelivr.net/" + strings.Repeat("x", MaxEventBytes)},
	})
	if _, err := ValidateEvent(oversized, id); err == nil {
		t.Fatal("ValidateEvent accepted oversized api-request event")
	}
}
