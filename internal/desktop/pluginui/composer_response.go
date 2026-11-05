package pluginui

import (
	_ "embed"
	"strings"
)

//go:embed composer_source.js
var composerSource string

// The shipped voice component consumes insertText's boolean before discarding
// its transcript. A native insertion is asynchronous; adapt these two concrete
// consumers so that only a successful native receipt consumes pending text.
func bridgeVoiceComposer(source []byte) ([]byte, bool) {
	text := string(source)
	if !strings.Contains(text, `id: "@deepseek-ai/dsh-experimental-client-ui-voice-input"`) {
		return source, false
	}
	finish := `if (!inputActions.insertText(result.value.text, active.span)) {`
	retry := "onClick: () => {\n\t\t\t\t\t\t\tif (inputActions.insertText(pending, inputActions.captureInsertion())) {"
	if strings.Count(text, finish) != 1 || strings.Count(text, retry) != 1 {
		return source, false
	}
	text = strings.Replace(text, finish, `if (!(inputActions.insertTextAsync ? await inputActions.insertTextAsync(result.value.text, active.span) : inputActions.insertText(result.value.text, active.span))) {`, 1)
	text = strings.Replace(text, retry, "onClick: async () => {\n\t\t\t\t\t\t\tconst span = inputActions.captureInsertion();\n\t\t\t\t\t\t\tif (inputActions.insertTextAsync ? await inputActions.insertTextAsync(pending, span) : inputActions.insertText(pending, span)) {", 1)
	return []byte(text), true
}

func bridgeConversationComposer(source []byte) ([]byte, bool) {
	text := string(source)
	module := `id: "@deepseek-ai/dsh-client-ui-conversation"`
	anchor := "\t\t\t\t\tconst shell = inputHub.shellFor(binding);"
	state := "\t\t\t\t\t\t\tinput: shell.state"
	actions := "\t\t\t\t\t\tprops: { inputActions: shell.actions }"
	if strings.Count(text, module) != 1 || strings.Count(text, anchor) != 1 || strings.Count(text, state) != 1 || strings.Count(text, actions) != 1 {
		return source, false
	}
	text = strings.Replace(text, anchor, anchor+"\n\t\t\t\t\tconst whaleNativeInput = window.__whaleNativeComposer?.provide(binding, shell);", 1)
	text = strings.Replace(text, state, "\t\t\t\t\t\t\tinput: whaleNativeInput?.state || shell.state", 1)
	text = strings.Replace(text, actions, "\t\t\t\t\t\tprops: { inputActions: whaleNativeInput?.actions || shell.actions }", 1)
	text, _ = bridgeConversationViews(text)
	text, _ = bridgeComposerBar(text)
	return []byte(composerSource + "\n" + text), true
}

func bridgeConversationViews(text string) (string, bool) {
	registry := "\t\t\tconst conversationViews = (0, _deepseek_ai_dsh_client_store.createSnapshotStore)(viewTabs());"
	selector := "\t\t\t\t\tselectView: (view) => {\n\t\t\t\t\t\tactivateView(sessionId, view);\n\t\t\t\t\t\tactions.setView(view);\n\t\t\t\t\t}"
	if strings.Count(text, registry) != 1 || strings.Count(text, selector) != 1 {
		return text, false
	}
	text = strings.Replace(text, registry, registry+`
            if (new URLSearchParams(location.search).get("desktopNativeSession") === "1") window.__whaleNativeConversation = {views: () => conversationViews.getSnapshot(), targets: () => uiConversation.views.entries().map(row => row.target), selectors:new Map()};`, 1)
	replacement := `                    selectView: ((select) => { window.__whaleNativeConversation?.selectors.set(sessionId, select); return select; })((view) => {
                        activateView(sessionId, view);
                        actions.setView(view);
                    })`
	return strings.Replace(text, selector, replacement, 1), true
}

// bridgeComposerBar preserves the actual public injected bar props. It wraps
// only the installed entry, not arbitrary plugin source or a guessed API.
func bridgeComposerBar(text string) (string, bool) {
	start := strings.Index(text, "\t\t\t\t\treturn {\n\t\t\t\t\t\tkeyboard: shell,")
	if start < 0 {
		return text, false
	}
	tail := text[start:]
	end := strings.Index(tail, "\n\t\t\t\t\t};\n\t\t\t\t}\n\t\t\t}, InputBar);")
	if end < 0 {
		return text, false
	}
	old := tail[:end+len("\n\t\t\t\t\t};")]
	wrapped := strings.Replace(old, "return {", "return (window.__whaleNativeComposer?.barProps || ((id, props) => props))(sessionId, {", 1)
	wrapped = strings.TrimSuffix(wrapped, "};") + "});"
	return text[:start] + wrapped + text[start+len(old):], true
}
