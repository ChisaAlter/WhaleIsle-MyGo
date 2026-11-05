package pluginui

import "strings"

// These are response-copy adapters for the two installed owners whose children
// mount on a native gesture. The original React state, props and unmount remain
// authoritative; the bridge does not manufacture a renderSlot owner.
func bridgePublicOwners(source []byte) ([]byte, bool) {
	text, changed := string(source), false
	if strings.Contains(text, `id: "@deepseek-ai/dsh-client-ui-settings-account"`) {
		anchor := `const [selectedAction, setSelectedAction] = (0, react.useState)(null);`
		if strings.Count(text, anchor) == 1 {
			text = strings.Replace(text, anchor, anchor+`
            (0, react.useEffect)(() => {
              if (new URLSearchParams(location.search).get('desktopNativeNavigation') !== '1') return;
              const controller = {select(id) { if (!launcherActions.some(row => row.id === id)) return false; setOpen(false); setSelectedAction(id); return true; }};
              window.__whalePublicLauncher = controller;
              window.dispatchEvent(new Event('whale-public-owners'));
              return () => { if (window.__whalePublicLauncher === controller) { delete window.__whalePublicLauncher; window.dispatchEvent(new Event('whale-public-owners')); } };
            }, [launcherActions]);`, 1)
			changed = true
		}
	}
	if strings.Contains(text, `id: "@deepseek-ai/dsh-client-ui-workspace"`) {
		anchor := "const [revealSessionId, setRevealSessionId] = (0, react.useState)(void 0);"
		if strings.Count(text, anchor) == 1 {
			text = strings.Replace(text, anchor, anchor+`
            (0, react.useEffect)(() => {
              if (new URLSearchParams(location.search).get('desktopNativeNavigation') !== '1') return;
              let query = '', archived = false;
              const controller = {apply(next) {
                if (query !== next.query) { query = next.query; setQuery(query); setSearchExpanded(query.length > 0); }
                if (archived !== next.archived) { archived = next.archived; setArchivedExpanded(archived); }
              }};
              window.__whalePublicWorkspace = controller;
              window.dispatchEvent(new Event('whale-public-owners'));
              return () => { if (window.__whalePublicWorkspace === controller) { delete window.__whalePublicWorkspace; window.dispatchEvent(new Event('whale-public-owners')); } };
            }, []);`, 1)
			changed = true
		}
		for _, row := range []struct{ name, id string }{{"SessionNodeItem", "node.id"}, {"ArchivedSessionNodeItem", "node.id"}, {"SearchResultItem", "result.id"}} {
			start := strings.Index(text, "function "+row.name+"(")
			if start < 0 {
				continue
			}
			end := strings.Index(text[start+9:], "\n\t\tfunction ")
			if end < 0 {
				continue
			}
			end += start + 9
			part := text[start:end]
			anchor := `const rowRef = (0, react.useRef)(null);`
			if strings.Count(part, anchor) != 1 || !strings.Contains(part, "setMenuOpen") {
				continue
			}
			injection := `
            (0, react.useEffect)(() => {
              if (new URLSearchParams(location.search).get('desktopNativeNavigation') !== '1') return;
              const rows = window.__whalePublicRows || (window.__whalePublicRows = new Map());
              const id = ` + row.id + `;
              let menu = false, hover = false;
              const controller = {apply(next) {
                if (menu !== next.menu) { menu = next.menu; setMenuOpen(menu); }
                if (hover !== next.hover) { hover = next.hover; const node = rowRef.current; if (node) node.dispatchEvent(new PointerEvent(hover ? 'pointerover' : 'pointerout', {bubbles:true, pointerId:1, pointerType:'mouse', relatedTarget:document.body})); }
              }};
              let owners = rows.get(id); if (!owners) rows.set(id, owners = new Set()); owners.add(controller);
              window.dispatchEvent(new Event('whale-public-owners'));
              return () => { owners.delete(controller); if (!owners.size) rows.delete(id); window.dispatchEvent(new Event('whale-public-owners')); };
            }, [` + row.id + `]);`
			part = strings.Replace(part, anchor, anchor+injection, 1)
			text = text[:start] + part + text[end:]
			changed = true
		}
	}
	return []byte(text), changed
}
