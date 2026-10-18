package agents

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestChildReadOnlyReasons(t *testing.T) {
	v := session.AgentNavigationView{Address: session.AgentSessionAddress{Mode: "continuable"}}
	unknown := childReadOnlyReason(v)
	v.ParentAvailabilityKnown = true
	offline := childReadOnlyReason(v)
	if unknown == "" || offline == "" || unknown == offline {
		t.Fatal("unknown parent conflated with offline")
	}
	v.ParentAvailable = true
	if childReadOnlyReason(v) != "" {
		t.Fatal("available continuation falsely read-only")
	}
	v.Address.Mode = "one-shot"
	one := childReadOnlyReason(v)
	v.Address.Mode = "unknown"
	other := childReadOnlyReason(v)
	if one == "" || other == "" || one == other {
		t.Fatal("mode-specific explanation missing")
	}
}
