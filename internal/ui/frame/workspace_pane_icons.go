package frame

import "github.com/egoist/mygo/ui"

var workspacePaneAddIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M8 3v10M3 8h10" stroke="currentColor"/></svg>`))
var workspacePaneResourcesIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M2 4.5 8 1.5l6 3v7l-6 3-6-3v-7Z M2 4.5l6 3 6-3M8 7.5v7" stroke="currentColor"/></svg>`))
var workspacePaneBrowserIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="6.25" stroke="currentColor"/><path d="M1.75 8h12.5M8 1.75c-3.2 3.1-3.2 9.4 0 12.5 3.2-3.1 3.2-9.4 0-12.5Z" stroke="currentColor"/></svg>`))
var workspacePaneTerminalIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><rect x="1.5" y="2.5" width="13" height="11" rx="1.5" stroke="currentColor"/><path d="m4 5.5 2.5 2.5L4 10.5M8.5 10.5H12" stroke="currentColor"/></svg>`))

// Original SidebarRight.tsx FullscreenGlyph/ExitFullscreenGlyph and
// ui-primitives/icons/index.tsx IconCloseFillArtwork, painted natively.
var workspacePaneFullscreenIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M2.33203 10.4054V13.1681C2.33229 13.444 2.55605 13.6681 2.83203 13.6681H5.49512V14.6681H2.83203C2.00376 14.6681 1.33229 13.9963 1.33203 13.1681V10.4054H2.33203ZM14.6689 13.1681C14.6687 13.996 13.9968 14.6676 13.1689 14.6681H10.4951V13.6681H13.1689C13.4445 13.6676 13.6687 13.4437 13.6689 13.1681V10.4054H14.6689V13.1681ZM13.1689 1.33118C13.9969 1.33163 14.6688 2.00315 14.6689 2.83118V5.4054H13.6689V2.83118C13.6688 2.55544 13.4446 2.33162 13.1689 2.33118H10.4951V1.33118H13.1689ZM5.49512 2.33118H2.83203C2.55598 2.33118 2.33218 2.55516 2.33203 2.83118V5.4054H1.33203V2.83118C1.33218 2.00288 2.00369 1.33118 2.83203 1.33118H5.49512V2.33118Z" fill="currentColor"/></svg>`))
var workspacePaneRestoreIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M9 2.5V6C9 6.26522 9.10536 6.51957 9.29289 6.70711C9.48043 6.89464 9.73478 7 10 7H13.5" stroke="currentColor"/><path d="M7 13.5V10C7 9.73478 6.89464 9.48043 6.70711 9.29289C6.51957 9.10536 6.26522 9 6 9H2.5" stroke="currentColor"/></svg>`))
var workspacePaneCloseIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M3.5 3.5L12.5 12.5M12.5 3.5L3.5 12.5" stroke="currentColor"/></svg>`))
