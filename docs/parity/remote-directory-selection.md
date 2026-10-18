# Remote directory and workspace mirror selection

Source contract: Electron `vendor/dsh-remote/plugin-src/client/RemoteFlowPane.jsx`, `client/api.js`, `lib/routes-fs.js`, and the `/dsh-remote/mirror` route in `lib/index.js`.

- GET `/api/dsh-remote/ls?path=...`: machine-scoped, no session binding; directories and Windows drive entries are navigable, files are displayed without selection actions.
- POST `/api/dsh-remote/mirror` with `{ "path": "..." }`: only the explicit create-mirror action sends this request. A nonempty returned `localMirror` is required; no invented RPC or client-side mirror path.
- The second explicit action adopts the returned mirror through the existing `applyWorkspacePath(..., false)` flow. It never makes a local directory or starts a file sync itself.
- Loading, empty directory, parent navigation, permission/missing errors and retry are visible. Closing/reopening the panel and refreshing the same machine retain successful browse and selected-mirror state. Navigating and failed requests retain the selected mirror. Machine identity changes and uncertain successful mutations clear stale directory state. Harness reattachment cancels and invalidates pending responses.

Verification is limited to local HTTP fixtures in `internal/store/remote/directory_test.go` and the existing remote store suite. No real server connection, remote directory read, credential change or mirror creation was performed. Windows build is performed once by the integration owner with the cumulative source tree.

Not included: remote mkdir, home shortcut, synchronization, transfer progress, changing a running session's binding, or replacing the existing machine editor.
