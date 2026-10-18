package pet

import (
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestPetTaskStatus(t *testing.T) {
	s := session.ViewState{ClientReady: true, SelectedID: "a<&", Items: []session.Item{{SessionID: "a<&", Title: "<script>bad</script>", Running: true}}}
	running := TaskFromSession(true, s)
	if running.State != "running" || running.SessionID != s.SelectedID {
		t.Fatalf("running: %+v", running)
	}
	s.Items[0].Running = false
	if got := TaskFromSession(true, s); got.State != "idle" {
		t.Fatal(got)
	}
	if got := TaskFromSession(false, s); got.State != "offline" || got.SessionID != "" {
		t.Fatal(got)
	}
	s.ListErr = "lost"
	if got := TaskFromSession(true, s); got.State != "unknown" || got.SessionID != "" {
		t.Fatal(got)
	}
	s.ListErr = ""
	s.SelectedID = "missing"
	if got := TaskFromSession(true, s); got.State != "unknown" {
		t.Fatal(got)
	}
	page := string(renderTaskPage([]byte(`{{STATUS}} {{TITLE}} {{OPEN}}`), running))
	if strings.Contains(page, "<script>") || !strings.Contains(page, "&lt;script&gt;") {
		t.Fatal(page)
	}
	if taskAction("whale-pet://localhost/open?session=a%3C%26", running) != "open" {
		t.Fatal("valid click lost")
	}
	for _, url := range []string{"whale-pet://localhost/open?session=other", "https://pet/hide", "whale-pet://localhost/hide?extra=1", "whale-pet://localhost/open?session=a%3C%26&extra=1", "whale-pet://user@pet/hide"} {
		if taskAction(url, running) != "" {
			t.Fatalf("accepted %s", url)
		}
	}
	if taskAction("whale-pet://localhost/hide", running) != "hide" {
		t.Fatal("hide lost")
	}
}
