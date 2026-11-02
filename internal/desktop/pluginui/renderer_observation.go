package pluginui

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type observedResponseBody struct {
	io.Reader
	io.Closer
}

const rendererModule = `id: "@deepseek-ai/dsh-client-ui-renderer"`
const rendererPrelude = `
		const whaleSessionEntries = new Map();
		const whaleSessionEntryIDs = new WeakMap();
		let whaleSessionEntrySerial = 0;
		function whaleSessionTurnTail(owner) {
			try {
				const tail = owner?.turn?.data?.get?.("turn-tail");
				return tail && Number.isSafeInteger(tail.turn) && tail.turn >= 0 && owner.turn.turn === tail.turn && Number.isSafeInteger(tail.seq) && tail.seq >= 0 ? tail : null;
			} catch { return null; }
		}
		function whaleSessionObserveEntry(Anchor, slotKey, entry, key, owner, content) {
			if (new URLSearchParams(location.search).get("desktopNativeSession") !== "1") return content;
			let id = whaleSessionEntryIDs.get(entry);
			if (id === undefined) {
				id = String(++whaleSessionEntrySerial);
				whaleSessionEntryIDs.set(entry, id);
				whaleSessionEntries.set(id, new WeakRef(entry));
				for (const [key, ref] of whaleSessionEntries) if (!ref.deref()) whaleSessionEntries.delete(key);
			}
			const tail = slotKey === "conversation.chat.turnTail" ? whaleSessionTurnTail(owner) : null;
			return react.createElement(Anchor, {key, "data-whale-slot-entry": id, "data-whale-slot-name": slotKey, "data-whale-turn-tail-turn": tail?.turn, "data-whale-turn-tail-end-seq": tail?.seq, style: {display: "contents"}}, content);
		}
		if (new URLSearchParams(location.search).get("desktopNativeSession") === "1") {
			Object.defineProperty(window, "__whaleSessionEntryOf", {configurable: true, value: (node, slot) => {
				if (!node?.isConnected || node.getAttribute("data-whale-slot-name") !== slot) return null;
				return whaleSessionEntries.get(node.getAttribute("data-whale-slot-entry"))?.deref() || null;
			}});
		}
`

// observeRenderer changes only the response copy of the known renderer. Its
// actual guarded entry remains the child, so elections, keys, hooks, errors and
// owner props are still chosen by the original renderer. Fallbacks never acquire
// an entry marker. Unknown versions are returned byte-for-byte unchanged.
func observeRenderer(source []byte) ([]byte, bool) {
	text := string(source)
	if strings.Count(text, rendererModule) != 1 || strings.Contains(text, "whaleSessionObserveEntry") {
		return source, false
	}
	start := strings.Index(text, "\t\tfunction SlotOutlet({ slotKey, ownerProps, opts, declaredInline }) {")
	end := strings.Index(text, "\t\t/** Root outlet:")
	if start < 0 || end <= start {
		return source, false
	}
	part := text[start:end]
	anchor := "\t\t\tconst guarded = (entry, key, owner = ownerProps) => {"
	dispatch := `return spec.scope === "session" ? (0, react_jsx_runtime.jsx)(StrictSessionEntry, {`
	finish := "\n\t\t\t\t}, key);\n\t\t\t};"
	if strings.Count(part, anchor) != 1 || strings.Count(part, dispatch) != 1 || strings.Count(part, finish) != 1 {
		return source, false
	}
	part = strings.Replace(part, dispatch, `return whaleSessionObserveEntry(Anchor, slotKey, entry, key, owner, spec.scope === "session" ? (0, react_jsx_runtime.jsx)(StrictSessionEntry, {`, 1)
	part = strings.Replace(part, finish, "\n\t\t\t\t}, key));\n\t\t\t};", 1)
	return []byte(text[:start] + rendererPrelude + part + text[end:]), true
}

// sessionRendererResponse is attached only to an explicitly opened Session seat.
// It preserves unknown renderer bytes and compressed responses it cannot decode.
func sessionRendererResponse(response *http.Response) error {
	return sessionResponse(response, true)
}

func sessionResponse(response *http.Response, observeEntries bool) error {
	if response.StatusCode != http.StatusOK || response.Request == nil || response.Request.Header.Get("Range") != "" || !strings.HasPrefix(response.Request.URL.Path, "/plugins/") {
		return nil
	}
	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "javascript") && !strings.Contains(contentType, "text/plain") {
		return nil
	}
	originalBody := response.Body
	raw, err := io.ReadAll(io.LimitReader(originalBody, 32*1024*1024+1))
	if err != nil {
		return err
	}
	if len(raw) > 32*1024*1024 {
		response.Body = &observedResponseBody{Reader: io.MultiReader(bytes.NewReader(raw), originalBody), Closer: originalBody}
		return nil
	}
	_ = originalBody.Close()
	restore := func() { response.Body = io.NopCloser(bytes.NewReader(raw)) }
	decoded := raw
	var reader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "", "identity":
	case "gzip":
		reader, err = gzip.NewReader(bytes.NewReader(raw))
	case "deflate":
		reader, err = zlib.NewReader(bytes.NewReader(raw))
	default:
		restore()
		return nil
	}
	if err != nil {
		restore()
		return nil
	}
	if reader != nil {
		decoded, err = io.ReadAll(io.LimitReader(reader, 32*1024*1024+1))
		_ = reader.Close()
		if err != nil || len(decoded) > 32*1024*1024 {
			restore()
			return nil
		}
	}
	observed, ok := decoded, false
	if observeEntries && response.Header.Get("Content-Security-Policy") == "" {
		observed, ok = observeRenderer(decoded)
	}
	withoutOnboarding, skippedOnboarding := suppressSessionOnboarding(observed)
	if skippedOnboarding {
		observed = withoutOnboarding
	}
	if !skippedOnboarding && bytes.Contains(decoded, []byte(`id: "@deepseek-ai/dsh-client-ui-settings-models"`)) {
		// The complete private response is rejected, including combo batches,
		// when the onboarding choice cannot be carried forward safely.
		observed = []byte(`window.__whaleSessionOnboardingIncompatible = true;`)
		response.Header.Set("X-Whale-Session-Incompatible", "1")
	} else if !ok && !skippedOnboarding {
		restore()
		return nil
	}
	response.Body = io.NopCloser(bytes.NewReader(observed))
	response.ContentLength = int64(len(observed))
	response.Header.Set("Content-Length", strconv.Itoa(len(observed)))
	response.Header.Del("Content-Encoding")
	response.Header.Del("ETag")
	response.Header.Del("Content-MD5")
	response.Header.Set("Cache-Control", "no-store")
	response.Header.Set("X-Whale-Session-Observation", "1")
	return nil
}
