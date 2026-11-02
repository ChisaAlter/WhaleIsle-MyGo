package pluginui

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionSuppressesWelcomeWithoutAcknowledgingOrInventingDesktopAPI(t *testing.T) {
	source := []byte(`id: "@deepseek-ai/dsh-client-ui-settings-models"; const credentialOnboarding = Config(payload === void 0 ? {} : payload).credentialOnboarding && !("dshDesktop" in globalThis); if (!("dshDesktop" in globalThis)) ctx.slots.inject("settings.onboarding", () => ctx.slots.register({`)
	if renderer := os.Getenv("WHALEISLE_PLUGIN_UI_RENDERER"); renderer != "" {
		var err error
		source, err = os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(renderer))), "ui-settings-models", "lib", "client.js"))
		if err != nil {
			t.Fatal(err)
		}
	}
	observed, ok := suppressSessionOnboarding(source)
	if !ok {
		t.Fatal("current Models onboarding policy not recognized")
	}
	if !bytes.Contains(observed, []byte(`window.__whaleSessionNoOnboarding = true`)) {
		t.Fatal("no-welcome policy did not settle")
	}
	for _, forbidden := range []string{`globalThis.dshDesktop =`, `window.dshDesktop =`, `welcomeNoticeVersion","2026`, `welcomeNoticeVersion', '2026`} {
		if bytes.Contains(observed, []byte(forbidden)) {
			t.Fatalf("invented acknowledgement or desktop API: %s", forbidden)
		}
	}
}

func TestUnknownOnboardingModuleCannotShowWelcome(t *testing.T) {
	source := []byte(`window.__ModuleLoader__.load({id: "@deepseek-ai/dsh-client-ui-settings-models", factory(){return futureUnverifiedWelcome}})`)
	r := &http.Response{StatusCode: 200, Request: &http.Request{URL: &url.URL{Path: "/plugins/"}}, Header: http.Header{"Content-Type": []string{"text/javascript"}}, Body: io.NopCloser(bytes.NewReader(source))}
	if err := sessionRendererResponse(r); err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r.Body)
	if !strings.Contains(string(out), "__whaleSessionOnboardingIncompatible") || strings.Contains(string(out), "futureUnverifiedWelcome") {
		t.Fatal("unknown onboarding could mount")
	}
}
