package main

import (
	"testing"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestConnectionBanner(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	ready := harness.Status{State: harness.Ready}
	if got := connectionBanner(ready, true, false); got != "" {
		t.Fatalf("ready mux: got %q", got)
	}
	if got := connectionBanner(ready, false, false); got != i18n.T(i18n.KeyChatConnConnecting) {
		t.Fatalf("connecting: got %q", got)
	}
	if got := connectionBanner(ready, false, true); got != i18n.T(i18n.KeyChatConnReconnecting) {
		t.Fatalf("reconnecting: got %q", got)
	}
	if got := connectionBanner(harness.Status{State: harness.Starting}, false, false); got != i18n.T(i18n.KeyBootStarting) {
		t.Fatalf("harness starting: got %q", got)
	}
}
