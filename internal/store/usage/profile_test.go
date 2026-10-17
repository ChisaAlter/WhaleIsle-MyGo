package usage

import (
	"encoding/json"
	"testing"
)

func TestParseProfileSummaryFlat(t *testing.T) {
	raw := json.RawMessage(`{"email":"a@b.c","displayName":"Ada","plan":"pro"}`)
	email, name, plan := ParseProfileSummary(raw)
	if email != "a@b.c" || name != "Ada" || plan != "pro" {
		t.Fatalf("got email=%q name=%q plan=%q", email, name, plan)
	}
}

func TestParseProfileSummaryNested(t *testing.T) {
	raw := json.RawMessage(`{"user":{"Email":"x@y.z","name":"Bob"},"account":{"tier":"plus"}}`)
	email, name, plan := ParseProfileSummary(raw)
	if email != "x@y.z" || name != "Bob" || plan != "plus" {
		t.Fatalf("got email=%q name=%q plan=%q", email, name, plan)
	}
}

func TestParseProfileSummaryEmpty(t *testing.T) {
	email, name, plan := ParseProfileSummary(nil)
	if email != "" || name != "" || plan != "" {
		t.Fatalf("expected empty, got %q %q %q", email, name, plan)
	}
}
