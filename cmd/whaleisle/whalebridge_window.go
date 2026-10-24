package main

import (
	"net/url"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/components"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/paths"
)

// WhaleBridge window (port of reference/electron/src/main/whalebridge-window.js:16):
// one 1120x780 window showing the local management page, refusing every other
// origin, and offering the account entry. The Go shell reuses the shared webview
// Host for the page instead of a second BrowserWindow, and the components
// start/stop/open lifecycle owns the process; the install action is unchanged.
const (
	whaleBridgeWindowWidth   = 1120
	whaleBridgeWindowHeight  = 780
	whaleBridgeToolbarHeight = 44
)

// whaleBridgeAccountsAnchor is the shell's own account-entry intent. The bridge
// management UI exposes no account route, so the shell marks the intent in the
// fragment and the injected script focuses the "供应商与账号" tab; a page without
// that tab is left untouched.
const whaleBridgeAccountsAnchor = "#accounts"

// whaleBridgeAccountsScript selects the management UI's account tab after the
// document is ready, and only when the shell asked for that entry.
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

// whaleBridgeWindow is the UI-thread-owned management window state.
type whaleBridgeWindow struct {
	win  *mygo.Window
	host *desktopwebview.Host
	url  string
	note string
}

// openWhaleBridgeWindow shows the management window for the running instance.
// The address is read from the component's own readiness state and validated by
// the components package; the window never loads a caller-supplied URL.
func (s *mainState) openWhaleBridgeWindow(parent *mygo.Window, accounts bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	existing := s.whaleBridgeWin
	quitting := s.quitting
	if existing == nil {
		s.whaleBridgeWin = &whaleBridgeWindow{}
		existing = s.whaleBridgeWin
	}
	s.mu.Unlock()
	if quitting {
		s.closeWhaleBridgeWindow()
		return
	}
	address, err := components.WhaleBridgeManagementAddress(paths.UserDataDir())
	if err != nil {
		if existing != nil && existing.win != nil && !existing.win.IsDestroyed() {
			s.closeWhaleBridgeWindow()
		}
		if parent != nil && !parent.IsDestroyed() {
			s.notifyUser(parent, components.FormatActionError("whalebridge", "open", err))
		}
		return
	}
	if existing.win != nil && !existing.win.IsDestroyed() {
		// One window per instance, like the reference module: a repeated entry
		// raises the existing window and re-applies the account entry.
		if accounts && existing.host != nil {
			_ = existing.host.Navigate(address + whaleBridgeAccountsAnchor)
		}
		existing.win.Focus()
		return
	}
	existing.url = address
	target := address
	if accounts {
		target = address + whaleBridgeAccountsAnchor
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:          resumeText("鲸桥 · WhaleBridge", "WhaleBridge"),
		Width:          whaleBridgeWindowWidth,
		Height:         whaleBridgeWindowHeight,
		MinWidth:       380,
		MinHeight:      520,
		UseContentSize: true,
		// Native whalebridge-window.js:18 sets autoHideMenuBar.
		AutoHideMenuBar: true,
		Content:         ui.View(existing.view),
	})
	existing.win = win
	if win == nil || win.IsDestroyed() {
		s.mu.Lock()
		if s.whaleBridgeWin == existing {
			s.whaleBridgeWin = nil
		}
		s.mu.Unlock()
		if parent != nil && !parent.IsDestroyed() {
			s.notifyUser(parent, resumeText("无法打开鲸桥管理窗口", "Could not open the WhaleBridge window"))
		}
		return
	}
	win.OnClosed(func() { s.closeWhaleBridgeWindow() })
	host, err := desktopwebview.NewHost(desktopwebview.HostOptions{
		Untrusted: true,
		Parent:    win,
		URL:       target,
		Title:     "WhaleBridge",
		TopInset:  whaleBridgeToolbarHeight,
		// Only this loopback management origin may load; a link to anything else
		// is handed to the browser, exactly as the reference window does.
		AllowNavigation: func(next string) bool {
			if sameManagementOrigin(next, address) {
				return true
			}
			if strings.HasPrefix(next, "http://") || strings.HasPrefix(next, "https://") || strings.HasPrefix(next, "mailto:") {
				_ = desktopwebview.OpenExternal(next)
			}
			return false
		},
		PreloadScript: whaleBridgeAccountsScript,
		OnExternalWindow: func(raw string) {
			_ = desktopwebview.OpenExternal(raw)
		},
		OnChange: func() { win.Update(func() {}) },
	})
	if err != nil {
		existing.note = resumeText("无法创建鲸桥页面：", "Could not create the WhaleBridge page: ") + err.Error()
		win.Update(func() {})
		return
	}
	existing.host = host
	existing.note = ""
	host.SetVisible(true)
	win.Focus()
}

// closeWhaleBridgeWindow destroys the management window and its webview. The
// bridge process keeps running unless the caller stops the component.
func (s *mainState) closeWhaleBridgeWindow() {
	if s == nil {
		return
	}
	s.mu.Lock()
	current := s.whaleBridgeWin
	s.whaleBridgeWin = nil
	s.mu.Unlock()
	if current == nil {
		return
	}
	host := current.host
	current.host = nil
	if host != nil {
		_ = host.Close()
	}
	win := current.win
	current.win = nil
	if win != nil && !win.IsDestroyed() {
		win.Destroy()
	}
}

// sameManagementOrigin reports whether target stays on the instance's origin.
// The validated management address carries scheme, loopback host and port only;
// its path and key never widen what the window may load.
func sameManagementOrigin(target, address string) bool {
	tu, err := url.Parse(target)
	if err != nil {
		return false
	}
	au, err := url.Parse(address)
	if err != nil || au.Host == "" {
		return false
	}
	return tu.Scheme == au.Scheme && strings.EqualFold(tu.Host, au.Host)
}

// view paints the native toolbar with the account entry. The webview covers
// everything below it, so the toolbar stays clickable while the page loads.
func (w *whaleBridgeWindow) view(c *ui.Context) {
	colors := c.Theme()
	ui.Column(c).Fill().Background(colors.Background).Children(func() {
		ui.Row(c).Fill().Height(whaleBridgeToolbarHeight).
			Padding(0, 12, 0, 12).AlignItems(ui.Center).Gap(8).
			Background(colors.Surface).Children(func() {
			ui.Text(c, resumeText("鲸桥管理", "WhaleBridge")).FontSize(13).TextColor(colors.Text).Grow(1)
			if w.host != nil {
				if ui.Button(c, resumeText("账号页", "Accounts")).Clicked() {
					_ = w.host.Navigate(w.url + whaleBridgeAccountsAnchor)
				}
				if ui.Button(c, resumeText("重新载入", "Reload")).Clicked() {
					w.host.Reload()
				}
			}
			if ui.Button(c, resumeText("关闭", "Close")).Clicked() {
				if w.win != nil && !w.win.IsDestroyed() {
					w.win.Close()
				}
			}
		})
		ui.Column(c).Fill().Grow(1).Padding(20, 20, 20, 20).Gap(8).Children(func() {
			if w.note != "" {
				ui.Text(c, w.note).FontSize(13).TextColor(colors.Text)
			} else {
				ui.Text(c, resumeText("正在打开鲸桥管理页…", "Opening the WhaleBridge management page…")).FontSize(13).TextColor(colors.TextMuted)
			}
		})
	})
}
