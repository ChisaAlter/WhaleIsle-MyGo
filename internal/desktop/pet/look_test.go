package pet

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLookLifecycle(t *testing.T) {
	var e lookEntry
	now := time.Now()
	ctx, serial := e.begin(now)
	if ctx == nil || !e.status.Busy {
		t.Fatal("begin")
	}
	if duplicate, _ := e.begin(now); duplicate != nil {
		t.Fatal("duplicate admitted")
	}
	e.lastCapture = now
	e.close()
	if ctx.Err() == nil || e.finish(serial, "stale reply") {
		t.Fatal("closed attempt completed")
	}
	if duringCooldown, _ := e.begin(now.Add(time.Second)); duringCooldown != nil {
		t.Fatal("cooldown bypassed")
	}
	ctx, next := e.begin(now.Add(4 * time.Second))
	if ctx == nil || !e.finish(next, "ready") || ctx.Err() == nil {
		t.Fatal("completion")
	}
}

func TestSendGlanceUsesOnlyWhaleRoute(t *testing.T) {
	var imageBytes bytes.Buffer
	if err := jpeg.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 20, 10)), nil); err != nil {
		t.Fatal(err)
	}
	var calls int
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/dsh-whale/pet/look" || r.Header.Get("Cookie") != "auth=fixture" {
			t.Error("wrong route or auth")
		}
		var request struct {
			Method  string
			Payload map[string]string
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Method != "pet/look" || request.Payload["provider"] != "vision-provider" || request.Payload["model"] != "image-model" || request.Payload["image"] == "" {
			t.Error("wrong payload")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "result": map[string]any{"ok": true, "value": map[string]any{"ok": true, "reply": "fixture reply"}}})
	}))
	defer host.Close()
	reply, err := SendGlance(context.Background(), host.URL, "auth=fixture", "vision-provider", "image-model", imageBytes.Bytes())
	if err != nil || reply != "fixture reply" || calls != 1 {
		t.Fatalf("%q %v calls=%d", reply, err, calls)
	}
	for _, origin := range []string{host.URL + "/path", host.URL + "?", host.URL + "#x"} {
		if _, err := SendGlance(context.Background(), origin, "", "p", "m", imageBytes.Bytes()); err == nil {
			t.Fatal("invalid origin")
		}
	}
	if _, err := SendGlance(context.Background(), host.URL, "", "", "", imageBytes.Bytes()); err == nil {
		t.Fatal("missing route")
	}
	if _, err := SendGlance(context.Background(), host.URL, "", "p", "m", []byte("not a screenshot")); err == nil {
		t.Fatal("invalid image")
	}
	if calls != 1 {
		t.Fatal("invalid input reached Host")
	}
}

func TestGlanceNoRedirectOrRawError(t *testing.T) {
	var frame bytes.Buffer
	_ = jpeg.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	var reached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	if _, err := SendGlance(context.Background(), redirect.URL, "auth=fixture", "p", "m", frame.Bytes()); err == nil || reached {
		t.Fatal("followed redirect")
	}
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"ok":true,"value":{"ok":false,"error":"private detail"}}}`))
	}))
	defer failed.Close()
	_, err := SendGlance(context.Background(), failed.URL, "", "p", "m", frame.Bytes())
	if err == nil || strings.Contains(err.Error(), "private detail") {
		t.Fatal("published Host failure detail")
	}
}

func TestLookActionAndReplyEscaping(t *testing.T) {
	if taskAction("whale-pet://localhost/look-screen", TaskStatus{}) != "look-screen" {
		t.Fatal("missing Look action")
	}
	for _, raw := range []string{"whale-pet://localhost/look-screen?", "whale-pet://localhost/look-screen?image=x", "whale-pet://localhost/look-screen#x", "https://localhost/look-screen"} {
		if taskAction(raw, TaskStatus{}) != "" {
			t.Fatalf("accepted %q", raw)
		}
	}
	body := renderLookPage([]byte("{{LOOK_STATUS}}"), LookStatus{Message: "<script>private</script>"})
	if strings.Contains(string(body), "<script>") {
		t.Fatal("reply HTML was not escaped")
	}
}
