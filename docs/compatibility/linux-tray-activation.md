# Linux tray activation (DSK04)

The original WhaleIsle tray registers `click` to show the main window; it does
not toggle a visible window off. MyGo's application already registers
`tray.OnClick(s.showMainWindow)`, and this change preserves that action.

Linux uses the AppIndicator backend and keeps its persistent menu. When the
loaded Ayatana library exposes the public `activate` signal, MyGo now routes
that signal to `TrayHandler.Clicked`. It probes the registered signal after
constructing the indicator rather than requiring a guessed version number.
The callback is allocated once and routes by live tray ID and native instance.

## Actual support boundary

- A compatible Ayatana library **and** a panel that sends the
  StatusNotifierItem `Activate` method can restore the window on primary
  activation while retaining the context menu.
- Older Ayatana/libappindicator libraries without the signal retain the
  menu-only path. The existing menu's open action remains available.
- Panels that choose to show a menu instead of issuing `Activate` remain
  menu-only, even with a compatible library.
- Ayatana's legacy GtkStatusIcon fallback still opens the menu from its
  `status_icon_activate` implementation. This change does not intercept it.
- Secondary activation (usually middle-click) is not treated as a left click.

This is capability-gated support, not a claim that all Linux desktops have
identical tray behavior.

## Evidence

- [Original tray at e9f584b](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/tray.js)
- [Ayatana public signal declaration](https://github.com/AyatanaIndicators/libayatana-appindicator/blob/main/src/app-indicator.h)
- [Ayatana signal registration, D-Bus dispatch and GtkStatusIcon fallback](https://github.com/AyatanaIndicators/libayatana-appindicator/blob/main/src/app-indicator.c)

## Verification

Focused Linux backend tests cover live-indicator routing, stale/mismatched
instances, and presence/absence of the public signal. They pass with
`CGO_ENABLED=0 go test ./internal/linux -run '^TestTrayActivation' -count=1`
from `third_party/mygo`. No desktop GUI session, real panel click, or cross-desktop
runtime validation was performed; the tests stub the native connection boundary.
