package main

import (
	"fmt"
	"github.com/egoist/mygo"
	"net/url"
	"strings"
	"whaleisle/internal/desktop/components"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

const whaleBridgeWindowWidth = 1120
const whaleBridgeWindowHeight = 780
const whaleBridgeAccountsAnchor = "#accounts"
const whaleBridgeAccountsScript = `(function () {
  if (location.hash !== '#accounts') return;
  function pick() {
    try {
      var tab = document.querySelector('[data-tab="providers"]');
      if (tab && tab.getAttribute('aria-selected') !== 'true') tab.click();
    } catch (e) {}
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () { setTimeout(pick, 300); }, { once: true });
  } else {
    setTimeout(pick, 300);
  }
})();`

// One URL window owns the full management viewport. Native hidden-title-bar
// controls overlay it, so no extra toolbar displaces the component document.
type whaleBridgeWindow struct {
	win    *mygo.Window
	url    string
	epoch  uint64
	failed uint64
}

const whaleBridgeChromeScript = `(function(){
 function mount(){
  if(document.getElementById('mygo-whalebridge-chrome'))return;
  var style=document.createElement('style');style.id='mygo-whalebridge-chrome';
  style.textContent='html{overflow:hidden;background:transparent!important}body{background:transparent!important;border-radius:20px;corner-shape:round}body::before{content:"";position:fixed;inset:0;z-index:-1;pointer-events:none;border-radius:20px;corner-shape:round;background:var(--dsw-alias-bg-base,#fff)}body::after{content:"";position:fixed;inset:0;z-index:2147483646;pointer-events:none;border-radius:20px;box-shadow:inset 0 0 0 1px var(--dsw-alias-border-l2,rgba(128,128,128,.4))}html[data-window-maximized] body,html[data-window-maximized] body::before{border-radius:0}html[data-window-maximized] body::after{display:none}#mygo-bridge-caption{position:fixed;left:0;right:120px;top:0;height:48px;z-index:10;--app-region:drag;-webkit-app-region:drag}';
  document.head.appendChild(style);var caption=document.createElement('div');caption.id='mygo-bridge-caption';caption.setAttribute('aria-hidden','true');document.body.appendChild(caption);
 }
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount,{once:true});else mount();
})();`

func (s *mainState) openWhaleBridgeWindow(parent *mygo.Window, accounts bool) {
	if s == nil {
		return
	}
	address, err := components.WhaleBridgeManagementAddress(paths.UserDataDir())
	if err != nil {
		if parent != nil {
			s.notifyUser(parent, components.FormatActionError("whalebridge", "open", err))
		}
		return
	}
	s.mu.Lock()
	current := s.whaleBridgeWin
	quitting := s.quitting
	s.mu.Unlock()
	if quitting {
		return
	}
	if current != nil && current.win != nil && !current.win.IsDestroyed() {
		if sameManagementOrigin(current.url, address) {
			if accounts {
				_ = current.win.Page().LoadURL(address + whaleBridgeAccountsAnchor)
			}
			current.win.Restore()
			current.win.Show()
			current.win.Focus()
			return
		}
		s.closeWhaleBridgeWindow()
	}
	target := address
	if accounts {
		target += whaleBridgeAccountsAnchor
	}
	current = &whaleBridgeWindow{url: address}
	win := mygo.NewWindow(mygo.WindowOptions{Title: i18n.T(i18n.KeyCmdWhalebridge), Width: 1120, Height: 780, MinWidth: 380, MinHeight: 520, Hidden: true,
		TitleBarStyle: mygo.TitleBarHidden, TitleBarHeight: 48, WindowsCaptionButtons: &mygo.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 8, Right: 8, CornerRadius: 8, GlyphSize: 12}, AutoHideMenuBar: true, Transparent: true, BackgroundColor: "#00000000",
		Page: mygo.PageOptions{DisableBindings: true, DevTools: mygo.DevToolsDisabled, PreloadScript: whaleBridgeAccountsScript + whaleBridgeChromeScript}})
	if win == nil || win.IsDestroyed() {
		if parent != nil {
			s.notifyUser(parent, i18n.T(i18n.KeyCmdCouldNotOpenTheWhalebridgeWindow))
		}
		return
	}
	current.win = win
	s.mu.Lock()
	s.whaleBridgeWin = current
	s.mu.Unlock()
	applyWindowIcon(win)
	pg := win.Page()
	syncShape := func() {
		script := "document.documentElement.toggleAttribute('data-window-maximized',"
		if win.IsMaximized() {
			script += "true"
		} else {
			script += "false"
		}
		script += ")"
		go func() { _, _ = pg.Eval(script) }()
	}
	win.OnMaximize(syncShape)
	win.OnUnmaximize(syncShape)
	pg.OnDidFinishLoad(syncShape)
	pg.SetPermissionHandler(desktopwebview.PermissionPolicy{NotificationsAllowed: systemNotificationsSupported}.Decide)
	pg.OnWillNavigate(func(e *mygo.NavigateEvent) {
		if !sameManagementOrigin(e.URL, address) {
			e.PreventDefault()
			if strings.HasPrefix(e.URL, "https://") || strings.HasPrefix(e.URL, "http://") || strings.HasPrefix(e.URL, "mailto:") {
				_ = desktopwebview.OpenExternal(e.URL)
			}
		}
	})
	pg.SetWindowOpenHandler(func(request mygo.WindowOpenRequest) *mygo.WindowOptions {
		if strings.HasPrefix(request.URL, "https://") || strings.HasPrefix(request.URL, "http://") {
			_ = desktopwebview.OpenExternal(request.URL)
		}
		return nil
	})
	pg.OnDidNavigate(func(string) { current.epoch++; current.failed = 0 })
	report := func(kind, message string) {
		epoch := current.epoch
		if current.failed == epoch+1 {
			return
		}
		current.failed = epoch + 1
		valid := func() bool {
			s.mu.Lock()
			same := s.whaleBridgeWin == current
			s.mu.Unlock()
			return same && !win.IsDestroyed() && current.epoch == epoch
		}
		s.onEmbeddedPageFailureFor(win, desktopwebview.PageFailure{Kind: kind, Message: message, Current: valid, Reload: func() {
			if valid() {
				pg.Reload()
			}
		}})
	}
	pg.OnDidFailLoad(func(f *mygo.LoadError) { report("load", fmt.Sprintf("Page load failed (engine code %d)", f.Code)) })
	pg.OnRenderProcessGone(func(reason string) { report("process", "Page process exited: "+reason) })
	win.OnReadyToShow(func() { win.Show(); win.Focus() })
	win.OnClosed(func() {
		s.mu.Lock()
		if s.whaleBridgeWin == current {
			s.whaleBridgeWin = nil
		}
		s.mu.Unlock()
	})
	if err := pg.LoadURL(target); err != nil {
		report("load", err.Error())
		win.Show()
	}
}
func (s *mainState) closeWhaleBridgeWindow() {
	s.mu.Lock()
	current := s.whaleBridgeWin
	s.whaleBridgeWin = nil
	s.mu.Unlock()
	if current != nil && current.win != nil && !current.win.IsDestroyed() {
		current.win.Destroy()
	}
}
func sameManagementOrigin(target, address string) bool {
	tu, err := url.Parse(target)
	if err != nil {
		return false
	}
	au, err := url.Parse(address)
	return err == nil && au.Host != "" && tu.Scheme == au.Scheme && strings.EqualFold(tu.Host, au.Host)
}
