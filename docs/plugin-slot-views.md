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

## Applying saved marketplace changes

A `restart-required` result remains visibly pending in the native marketplace.
The explicit “Restart runtime and apply” action uses the existing import and
Host task-protection checks; it never repeats installation. Cancellation or a
failed restart keeps the pending change. Only a newer Supervisor generation
that is actually Ready clears it and refreshes inventory, including same-port
restarts. Existing `applied` changes continue to use the original client HMR
reconciliation and Cordis owner disposal.

When a Host restarts while the plugin panel is still selected, its old window
and authorization are destroyed, then the logical plugin URL is restored on
the new Ready generation. Closing the view or navigating elsewhere cancels
that restoration.

## Configuration owners and lifecycle results

The view picker also walks declared children of main panels. Contributions to
`plugins.bundle.config` and `plugins.row.config` open their bundle through the
original public `pluginNavigation.openBundle` service (or the owning main
panel if that service is unavailable). Row configuration continues through the
original bundle's configure controls. Official `plugins.item` forms open via
their original, uniquely identified accessible card title. The owner therefore
retains revision-aware form writes, detail actions, badges, sections and row
lifecycle controls. A removed contribution cancels pending picker navigation.

Native Settings and Installed now take enabled state and write restrictions from
Host bundle inventory. They do not optimistically edit local DisabledPlugins or
retry a package name as a plugin-entry identifier. Both surfaces share the
generation-bound marketplace operation gate and show errors, overridden changes,
pending restarts and failed restart state. Read-only, unidentified, disconnected
or stale inventory cannot expose bundle mutation controls; callbacks recheck the
inventory before issuing a write. Inspection uses the same generation guard.

Source contract: upstream ui-plugin-manager `slot-contract.ts`, `index.ts`,
`manager-store.ts` and `PluginManagerPage.tsx` at
`90465b67ec16765631d54c195af19d9ff3c3a770`.

The adapter fixture was executed in a JavaScript-only mock context, covering
main, settings, remote, sidebar, nested bundle/row/official configuration,
single activation and disposal. Go control tests are included but were not run
in this disk-full execution environment; no real Host or plugin was executed.
