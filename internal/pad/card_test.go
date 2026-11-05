package pad

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	store "whaleisle/internal/store/pad"
)

func TestPadCardUsesReleaseRuntimeAndOpaqueNativeBody(t *testing.T) {
	bundle, err := os.ReadFile(filepath.Join("..", "..", "runtime", "vendor", "dsh-pad", "lib", "client.js"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := frameRuntime(string(bundle))
	if err != nil {
		t.Fatal(err)
	}
	for _, api := range []string{"run: function", "state:", "history: function", "sparkline:", "window.dsh = dsh"} {
		if !strings.Contains(runtime, api) {
			t.Fatal(api)
		}
	}
	if _, err := frameRuntime("bad bundle"); err == nil {
		t.Fatal("unsupported runtime silently accepted")
	}
	p := store.Panel{ID: "qa", Version: 1, Definition: store.Definition{Title: "qa", HTML: "<div id=out></div>", Script: "dsh.onData(function(d){document.getElementById('out').textContent='ok'})", Height: json.RawMessage(`"auto"`), Sources: []store.Source{}}, Data: store.Data{Sources: map[string]store.Reading{}}}
	ready := make(chan struct{}, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		raw, _ := json.Marshal(map[string]any{"type": "panels", "panels": []store.Panel{p}})
		fmt.Fprintf(w, "data: %s\n\n", raw)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	s, err := store.Open(t.TempDir(), func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()
	client, _ := store.NewClient(server.URL, "secret-host-cookie")
	s.Attach(client, 9)
	deadline := time.After(2 * time.Second)
	for !s.Snapshot().Loaded {
		select {
		case <-ready:
		case <-deadline:
			t.Fatal("panel not received")
		}
	}
	d := &Desktop{Store: s, runtime: runtime, appearance: Appearance{CSS: ":root{--pad-fg:#fff}", Mode: "dark", Lang: "zh-CN"}}
	s.UpdateLayout(func(l *store.Layout) { l.Expanded = true; l.Width = 352 })
	if d.ReservedWidth(1279) != 0 || d.ReservedWidth(1280) != 364 {
		t.Fatal("narrow overlay reserved content or wide dock did not", d.ReservedWidth(1279), d.ReservedWidth(1280))
	}
	s.UpdateLayout(func(l *store.Layout) { l.Dock = &store.PointXY{X: 40, Y: 64} })
	if d.ReservedWidth(1440) != 0 {
		t.Fatal("floating dock reserved a main-content column")
	}
	s.UpdateLayout(func(l *store.Layout) { l.Dock = nil; l.Expanded = false })
	if d.ReservedWidth(1440) != 0 {
		t.Fatal("folded Pad reserved content")
	}
	c := &card{desktop: d, key: "qa", id: "qa", token: "opaque-frame-token", version: 1, generation: 9}
	document, err := c.document()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document, `sandbox="allow-scripts"`) || strings.Contains(document, "allow-same-origin") || strings.Contains(document, "secret-host-cookie") || strings.Contains(document, "__ModuleLoader__") {
		t.Fatal("card acquired full Host UI or credentials")
	}
	scripts := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindAllStringSubmatch(document, -1)
	if len(scripts) != 1 {
		t.Fatalf("outer shell contains %d scripts", len(scripts))
	}
	node := filepath.Join("..", "..", "runtime", "node", "node.exe")
	if _, err := os.Stat(node); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "pad-shell.js")
	if err := os.WriteFile(script, []byte(scripts[0][1]), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(node, "--check", script).CombinedOutput(); err != nil {
		t.Fatalf("shell syntax: %v %s", err, out)
	}
}

func TestFrameEndTagsAreEscapedCaseInsensitively(t *testing.T) {
	for _, source := range []string{`</style><script>unsafe</script>`, `</STYLE><SCRIPT>unsafe</SCRIPT>`, `</StYlE><ScRiPt>unsafe</ScRiPt>`} {
		if escaped := escapeFrameEndTag(source); regexp.MustCompile(`(?i)</(?:style|script)`).MatchString(escaped) || !strings.Contains(escaped, `<\/`) {
			t.Fatalf("authored source can terminate its sandbox element: %q", escaped)
		}
	}
}
