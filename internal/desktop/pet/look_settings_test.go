package pet

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/petlook"
)

func TestPetLookSelectionContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)
	seed := `{"live2dPet":{"settings":{"lookProvider":"legacy","lookModel":"keep"},"growth":{"points":99}},"globalVision":{"provider":"global","model":"global-model"}}`
	if err := os.WriteFile(path, []byte(seed), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if b := s.DesktopPet().Behavior; b.LookProvider != "" || b.LookModel != "" {
		t.Fatal("must default off, not inherit a global or legacy route")
	}
	cat := petlook.Catalog{Connected: true, Routes: []petlook.Route{{Provider: "chosen", Model: "image"}, {Provider: "other", Model: "image"}}}
	p := &Companion{Store: s, SkinID: "unsaved", Behavior: config.DefaultPetBehavior()}
	p.Behavior.Personality = "genki"
	if !p.SaveLookRoute("chosen", "image", cat) {
		t.Fatal(p.Error)
	}
	if p.Behavior.Personality != "genki" || p.SkinID != "unsaved" || s.DesktopPet().Behavior.Personality != "natural" {
		t.Fatal("selection wrote unrelated drafts")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(before, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"live2dPet", "globalVision"} {
		var orig map[string]json.RawMessage
		_ = json.Unmarshal([]byte(seed), &orig)
		var a, b bytes.Buffer
		_ = json.Compact(&a, orig[key])
		_ = json.Compact(&b, obj[key])
		if !bytes.Equal(a.Bytes(), b.Bytes()) {
			t.Fatalf("changed %s", key)
		}
	}
	for _, invalid := range []petlook.Catalog{{}, {Connected: true, Loading: true, Routes: cat.Routes}, {Connected: true, Err: "failed", Routes: cat.Routes}, {Connected: true}} {
		if petlook.Resolve("chosen", "image", invalid) {
			t.Fatal("unavailable catalog admitted")
		}
		if p.SaveLookRoute("other", "image", invalid) {
			t.Fatal("unavailable selection committed")
		}
	}
	for _, route := range []petlook.Route{{Provider: "chosen"}, {Provider: "missing", Model: "image"}, {Provider: "chosen", Model: "text"}} {
		if petlook.Resolve(route.Provider, route.Model, cat) || p.SaveLookRoute(route.Provider, route.Model, cat) {
			t.Fatalf("invalid route admitted: %+v", route)
		}
	}
	if p.Behavior.LookProvider != "chosen" || s.DesktopPet().Behavior.LookProvider != "chosen" {
		t.Fatal("failure lost saved route")
	}
	// Installed desktop-live2d lookAvailable depends on model, not provider.
	// A model-only route is explicit legacy routing, never a catalog fallback.
	if petlook.Resolve("", "image", cat) || !p.SaveLookRoute("", "image", petlook.Catalog{}) || s.DesktopPet().Behavior.LookProvider != "" || s.DesktopPet().Behavior.LookModel != "image" {
		t.Fatal("explicit legacy model-only route lost")
	}
	if !p.SaveLookRoute("chosen", "image", cat) {
		t.Fatal(p.Error)
	}
	// A route edit invalidates a pending capture. Even if a late frame reaches
	// the sender, it cannot use the former context or silently choose the new one.
	ctx, serial := p.look.begin(time.Now())
	if ctx == nil || !p.SaveLookRoute("other", "image", cat) || ctx.Err() == nil || p.look.finish(serial, "late") {
		t.Fatal("route change did not invalidate the old attempt")
	}
	var frame bytes.Buffer
	if err := jpeg.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	calls := 0
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request struct{ Payload map[string]string }
		_ = json.NewDecoder(r.Body).Decode(&request)
		if r.URL.Path != "/dsh-whale/pet/look" || request.Payload["provider"] != "other" || request.Payload["model"] != "image" {
			t.Error("wrong screenshot target")
		}
		http.Error(w, "failed", http.StatusServiceUnavailable)
	}))
	defer host.Close()
	if _, err := SendGlance(ctx, host.URL, "fixture", "chosen", "image", frame.Bytes()); err == nil || calls != 0 {
		t.Fatal("late cancelled frame was sent")
	}
	if _, err := SendGlance(context.Background(), host.URL, "fixture", "other", "image", frame.Bytes()); err == nil || calls != 1 {
		t.Fatal("failure retried/fell back or did not reach exact target")
	}
	// Reset clears both new fields, cancels the current attempt and preserves
	// the original all-or-nothing configuration semantics.
	ctx, _ = p.look.begin(time.Now())
	if !p.ResetAppearanceBehavior() || ctx.Err() == nil {
		t.Fatal("reset did not cancel look")
	}
	if got := s.DesktopPet().Behavior; got.LookProvider != "" || got.LookModel != "" {
		t.Fatal("reset did not turn Look off")
	}
	if !p.SaveLookRoute("chosen", "image", cat) || !p.SaveLookRoute("", "", petlook.Catalog{}) {
		t.Fatal("explicit Off must work offline")
	}
	if err := os.WriteFile(path, []byte(`{"changed":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	previous := p.Behavior
	if p.SaveLookRoute("chosen", "image", cat) || p.Behavior != previous {
		t.Fatal("conflict replaced draft")
	}
	var malformed config.PetBehavior
	if err := json.Unmarshal([]byte(`{"lookProvider":false,"lookModel":"  image  "}`), &malformed); err != nil {
		t.Fatal(err)
	}
	if malformed.LookProvider != "" || malformed.LookModel != "image" {
		t.Fatal("malformed fields damaged siblings")
	}
	if got := petlook.NormalizeID("  " + strings.Repeat("a", 129) + "  "); len(got) != 128 {
		t.Fatal("ID cap")
	}
}
