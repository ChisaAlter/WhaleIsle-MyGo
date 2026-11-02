package pluginui

import (
	"net/http"
	"strings"
)

func (o *Owner) sessionResponse(response *http.Response) error {
	err := sessionRendererResponse(response)
	o.acceptSessionResponse(response)
	return err
}

func (o *Owner) acceptSessionResponse(response *http.Response) {
	if response.Header.Get("X-Whale-Session-Incompatible") != "1" {
		return
	}
	o.mu.Lock()
	o.incompatible = true
	o.state.Subject = nil
	o.state.Actions = nil
	o.state.SessionControls = nil
	o.state.Outlets = nil
	o.state.Surface = ""
	o.state.SessionSurfaceHandle = ""
	o.state.SessionSurfaceRequest = ""
	o.state.Unavailable = "session-version-incompatible"
	o.mu.Unlock()
	o.Hide()
	o.notify()
}

// suppressSessionOnboarding carries the native application's existing no-welcome
// choice into its binding-free local Session view. It neither acknowledges the
// notice nor changes stored credentials, provider configuration or permissions.
func suppressSessionOnboarding(source []byte) ([]byte, bool) {
	text := string(source)
	module := `id: "@deepseek-ai/dsh-client-ui-settings-models"`
	auto := `const credentialOnboarding = Config(payload === void 0 ? {} : payload).credentialOnboarding && !("dshDesktop" in globalThis);`
	welcome := `if (!("dshDesktop" in globalThis)) ctx.slots.inject("settings.onboarding", () => ctx.slots.register({`
	if strings.Count(text, module) != 1 || strings.Count(text, auto) != 1 || strings.Count(text, welcome) != 1 {
		return source, false
	}
	text = strings.Replace(text, auto, `const credentialOnboarding = new URLSearchParams(location.search).get("desktopNativeSession") !== "1" && Config(payload === void 0 ? {} : payload).credentialOnboarding && !("dshDesktop" in globalThis);`, 1)
	text = strings.Replace(text, welcome, `if (new URLSearchParams(location.search).get("desktopNativeSession") !== "1" && !("dshDesktop" in globalThis)) ctx.slots.inject("settings.onboarding", () => ctx.slots.register({`, 1)
	text += "\nif (new URLSearchParams(location.search).get(\"desktopNativeSession\") === \"1\") window.__whaleSessionNoOnboarding = true;\n"
	return []byte(text), true
}
