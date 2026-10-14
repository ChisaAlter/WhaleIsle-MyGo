# Plugin view consumption

The native Plugins settings page can open an installed plugin's frontend view.
`?desktopPlugin=<npm package>` is consumed by the application-owned
`whaleisle-desktop-plugin-views` client overlay prepared under the active profile.
It is not an upstream Harness URL feature and does not remove the Host shell.

The adapter reads the live SlotRegistry's winning entries and matches their
`registrant` package. It follows the existing owner instead of copying plugin
components or bypassing renderSlot authorization:

- `main`: `layout.selectPanel(key)`
- `settings.section`: `settingsNavigation.open(id)`
- `settings.remote.tab`: open its Remote settings owner, then activate its
  existing `data-dsh-settings-remote-tab` control
- `sidebar.page`: activate the uniquely matching existing ARIA tab, using the
  label from its `sidebar.nav.tab` declaration
- Nested Settings contributions: walk declared children and open the owning
  section without reconstructing plugin props

One initial activation is performed. The chooser follows slot registration and
unload changes, but does not steal navigation after Back/Close or a plugin's own
session transition. Contextual-only plugins show an explicit message; their
session-bound views still require the appropriate conversation context. This is
not a native replacement for every plugin-specific desktop API.

Plugin pages have all MyGo bindings and application events disabled. A per-window
loopback proxy targets only the exact active Host origin. It uses the existing
Host session cookie only in upstream request headers, and keeps it out of browser
JavaScript, URLs and logs. The proxy uses an independent random, HttpOnly,
SameSite=Strict browser capability, never persists credentials, rejects foreign
Host/Origin requests, blocks off-Host redirects, and dies on window close or Host
session change (including upgraded WebSocket connections). The native chrome
shows/copies/restores the original logical Host URL, not the proxy capability.

The overlay lives in the desktop profile and does not modify user or vendor
packages. Host authentication and the Cordis runtime remain the authorities for
plugin RPC and slot ownership.

Verification: one temporary mock fixture covers live view activation/disposal,
HTTP and WebSocket forwarding, denied cross-origin requests/redirects, changed
Host credentials and complete gateway shutdown. No real Host, account, workspace
or plugin state is used.
