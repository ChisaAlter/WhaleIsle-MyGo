package mygo

import (
	"encoding/json"
	"runtime"
	"strings"
	"time"
)

// WebApplicationShortcut is an application-owned fallback for a web page.
// Browser/editor commands must stay in the page. Registration never grants a
// page a native binding: a matching real native key is required for each call.
type WebApplicationShortcut struct {
	Key              string
	Ctrl, Alt, Shift bool
	Run              func()
}
type webShortcutKey struct {
	VK   uint32
	Mods uint8
}
type webShortcutState struct {
	parent                    *Window
	bindings                  map[webShortcutKey]func()
	pending                   webShortcutKey
	pendingRun                func()
	pendingUntil              time.Time
	revision, pendingRevision uint64
}

const webShortcutPrefix = "mygo:web-shortcut:"

// SetWebApplicationShortcuts replaces only the fallback application registry.
// Windows WebView2 currently provides the required native physical-key event;
// other backends fail closed and continue their normal page keyboard handling.
func (w *Window) SetWebApplicationShortcuts(bindings []WebApplicationShortcut) {
	copyBindings := append([]WebApplicationShortcut(nil), bindings...)
	onMain(func() {
		if w.webShortcuts == nil {
			w.webShortcuts = &webShortcutState{}
		}
		next := map[webShortcutKey]func(){}
		for _, b := range copyBindings {
			vk := webShortcutVirtualKey(b.Key)
			if vk == 0 || !b.Ctrl || b.Run == nil {
				continue
			}
			mods := uint8(1)
			if b.Alt {
				mods |= 2
			}
			if b.Shift {
				mods |= 4
			}
			next[webShortcutKey{vk, mods}] = b.Run
		}
		w.webShortcuts.bindings = next
		w.webShortcuts.revision++
	})
}

// UseWebApplicationShortcuts opts an ordinary browser/plugin page into its
// parent's fallback registry. It injects no bridge, secret or Host RPC. Call
// before navigation. Local isolated document viewers must not opt in.
func (p *Page) UseWebApplicationShortcuts(parent *Window) bool {
	if p == nil || parent == nil || runtime.GOOS != "windows" {
		return false
	}
	onMain(func() {
		p.w.webShortcuts = &webShortcutState{parent: parent}
		reset := func() {
			if p.w.webShortcuts != nil {
				p.w.webShortcuts.pendingRun = nil
			}
		}
		p.w.OnBlur(reset)
		p.OnWillNavigate(func(*NavigateEvent) { reset() })
		p.w.OnClosed(reset)
		install := func() { p.Eval(webShortcutScript) }
		p.OnDidFinishLoad(install)
		install()
	})
	return true
}
func webShortcutVirtualKey(key string) uint32 {
	key = strings.ToUpper(key)
	if len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9')) {
		return uint32(key[0])
	}
	names := map[string]uint32{"ENTER": 13, ",": 188, ".": 190, "/": 191, ";": 186, "'": 222, "[": 219, "]": 221, "\\": 220, "`": 192, "-": 189, "=": 187}
	if v := names[key]; v != 0 {
		return v
	}
	for n := 1; n <= 12; n++ {
		if key == "F"+webShortcutNumber(n) {
			return uint32(111 + n)
		}
	}
	return 0
}
func webShortcutNumber(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return "1" + string(rune('0'+n-10))
}

// Optional backend event, deliberately not a method callable by the renderer.
// It does not mark the native event handled. IME, editors and page registries
// receive their original event unchanged, and can veto the fallback.
func (h *windowHandler) WebApplicationKey(vk uint32, ctrl, alt, shift, super, repeat bool) {
	s := h.w.webShortcuts
	if s == nil || s.parent == nil {
		return
	}
	s.pendingRun = nil
	if !ctrl || super || repeat || s.parent.IsDestroyed() || s.parent.webShortcuts == nil {
		return
	}
	mods := uint8(1)
	if alt {
		mods |= 2
	}
	if shift {
		mods |= 4
	}
	key := webShortcutKey{vk, mods}
	run := s.parent.webShortcuts.bindings[key]
	if run == nil {
		return
	}
	s.pending, s.pendingRun, s.pendingUntil = key, run, time.Now().Add(350*time.Millisecond)
	s.pendingRevision = s.parent.webShortcuts.revision
}
func (w *Window) handleWebShortcutMessage(raw string) bool {
	body, ok := strings.CutPrefix(raw, webShortcutPrefix)
	if !ok {
		return false
	}
	s := w.webShortcuts
	if s == nil || s.pendingRun == nil {
		return true
	}
	var key webShortcutKey
	if json.Unmarshal([]byte(body), &key) != nil || key != s.pending {
		return true
	}
	run := s.pendingRun
	s.pendingRun = nil
	if time.Now().After(s.pendingUntil) || w.IsDestroyed() || s.parent == nil || s.parent.IsDestroyed() {
		return true
	}
	// Rebinding/removal during propagation must not retain the old command.
	if s.parent.webShortcuts == nil || s.parent.webShortcuts.bindings[key] == nil || s.pendingRevision != s.parent.webShortcuts.revision {
		return true
	}
	run()
	return true
}

// Reporting happens after event propagation. A plugin's own registry keeps
// first refusal, even if its listener was installed later. Browser defaults are
// never canceled here; the application must exclude browser-owned chords.
const webShortcutScript = `(function(){
 if(window.__mygoApplicationShortcutObserver)return;
 window.__mygoApplicationShortcutObserver=true;
 let composing=false;
 window.addEventListener('compositionstart',()=>{composing=true},true);
 window.addEventListener('compositionend',()=>{composing=false},true);
 function owns(e){
  if(composing||e.isComposing||e.keyCode===229||e.repeat||!e.isTrusted)return true;
  if(document.querySelector('dialog[open],[aria-modal="true"]'))return true;
  const path=e.composedPath?e.composedPath():[e.target];
  return path.concat([document.activeElement]).some(n=>n&&n.nodeType===1&&(
   n.isContentEditable||/^(INPUT|TEXTAREA|SELECT|IFRAME)$/.test(n.tagName)||
   (n.matches&&n.matches('[role="textbox"],[role="combobox"],[role="searchbox"],.xterm,.terminal,[data-terminal],[data-shortcut-owner]'))));
 }
 window.addEventListener('keydown',e=>{
  if(!e.ctrlKey||e.metaKey||owns(e))return;
  const VK=e.keyCode,Mods=1|(e.altKey?2:0)|(e.shiftKey?4:0);
  setTimeout(()=>{if(!e.defaultPrevented&&!owns(e))window.chrome?.webview?.postMessage('mygo:web-shortcut:'+JSON.stringify({VK,Mods}));},0);
 });
})();`
