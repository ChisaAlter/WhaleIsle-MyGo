package pluginui

import (
	"net/http"
	"strings"
)

// Native Pad owns the dock and card lifecycle. Do not start a second PadClient
// inside a hidden navigation/Session page; ordinary full Host pages are intact.
// A navigation page carrying whalePadOriginal opts into hosting the shipped
// overlay itself while the native pad desktop stays retired.
func suppressNativePad(source []byte) ([]byte, bool) {
	text := string(source)
	module := `window.__ModuleLoader__.load({ id: "dsh-pad", factory:`
	begin := strings.Index(text, module)
	if begin < 0 {
		return source, false
	}
	end := strings.Index(text[begin:], "return module.exports; } });")
	if end < 0 {
		return source, false
	}
	end += begin
	part := text[begin:end]
	if !strings.Contains(part, `var NS = "dsh-pad";`) || strings.Count(part, "function start(ctx) {") != 1 || !strings.Contains(part, "const client = new PadClient();") {
		return source, false
	}
	part = strings.Replace(part, "function start(ctx) {", `function start(ctx) {
  if (['desktopNativeNavigation','desktopNativeSession','desktopNativePlugins'].some(key => new URLSearchParams(location.search).get(key) === '1') && new URLSearchParams(location.search).get('whalePadOriginal') !== '1') return;
  if (new URLSearchParams(location.search).get('whalePadOriginal') === '1' && typeof composer === 'object' && composer && typeof composer.attach === 'function') {
    composer.attach({ setDraft: text => window.__whalePadComposer?.(text, false), submit() {}, captureInsertion: () => ({}), insertText: text => Boolean(window.__whalePadComposer?.(text, true)) });
    composer.attach = () => () => {};
  }`, 1)
	text = text[:begin] + part + text[end:]
	return []byte(text), true
}

func bridgeWorkspaceActionChanges(source []byte) ([]byte, bool) {
	text := string(source)
	if !strings.Contains(text, `id: "@deepseek-ai/dsh-client-ui-workspace"`) {
		return source, false
	}
	for _, call := range []string{"this.sessionActions.set(owner, actions);", "this.sessionActions.delete(owner);"} {
		if strings.Count(text, call) != 1 {
			return source, false
		}
	}
	for _, call := range []string{"this.sessionActions.set(owner, actions);", "this.sessionActions.delete(owner);"} {
		text = strings.Replace(text, call, call+` window.dispatchEvent(new Event('whale-workspace-actions'));`, 1)
	}
	return []byte(text), true
}

// The original Models section supplies complete directory/credential facts for
// an adopted native catalog draft. It remains the sole renderSlot owner; this
// only adds the dormant row which its ordinary add-card would otherwise own.
func bridgeModelDrafts(source []byte) ([]byte, bool) {
	text := string(source)
	anchor := `const addRow = draft === void 0 ? void 0 : state.rows.find((row) => row.entry.provider === draft.target.provider);`
	children := `className: ModelsSection_module_css_default["section"],
				children: [`
	if !strings.Contains(text, `id: "@deepseek-ai/dsh-client-ui-settings-models"`) || strings.Count(text, anchor) != 1 || strings.Count(text, children) != 1 {
		return source, false
	}
	text = strings.Replace(text, anchor, anchor+`
            const whaleNativeModels = new URLSearchParams(location.search).get('desktopNativeNavigation') === '1';
            const [whaleModelTargets, setWhaleModelTargets] = (0, react.useState)(() => window.__whaleModelTargets || []);
            (0, react.useEffect)(() => {
              if (!whaleNativeModels) return;
              const update = () => setWhaleModelTargets(window.__whaleModelTargets || []);
              window.addEventListener('whale-model-targets', update); update();
              return () => window.removeEventListener('whale-model-targets', update);
            }, []);`, 1)
	text = strings.Replace(text, children, children+`
                  ...(whaleNativeModels ? state.rows.filter(row => !row.configured && whaleModelTargets.includes(row.entry.provider) && addRow?.entry.provider !== row.entry.provider).map(row => (0, react_jsx_runtime.jsx)('div', {children: renderSlot('settings.models.provider-card', {provider: row.entry, configured: row.configured, keyConfigured: keyConfiguredOf(row)}, {entryKey: row.entry.settingsNs})}, 'native-draft:' + row.entry.provider)) : []),`, 1)
	return []byte(text), true
}

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
	o.retireWorkspaceOperationsLocked()
	o.retireRightPaneOpensLocked()
	o.state.Subject = nil
	o.state.Actions = nil
	o.state.SessionControls = nil
	o.state.Outlets = nil
	o.state.RightPaneTabs = nil
	o.state.RightPaneKinds = nil
	o.state.Documents = nil
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
	text = strings.Replace(text, auto, `const credentialOnboarding = new URLSearchParams(location.search).get("desktopNativeSession") !== "1" && new URLSearchParams(location.search).get("desktopNativeNavigation") !== "1" && Config(payload === void 0 ? {} : payload).credentialOnboarding && !("dshDesktop" in globalThis);`, 1)
	text = strings.Replace(text, welcome, `if (new URLSearchParams(location.search).get("desktopNativeSession") !== "1" && new URLSearchParams(location.search).get("desktopNativeNavigation") !== "1" && !("dshDesktop" in globalThis)) ctx.slots.inject("settings.onboarding", () => ctx.slots.register({`, 1)
	text += "\nif (new URLSearchParams(location.search).get(\"desktopNativeSession\") === \"1\") window.__whaleSessionNoOnboarding = true;\n"
	return []byte(text), true
}
