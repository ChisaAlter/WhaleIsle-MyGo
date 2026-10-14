# MCP settings edit/remove parity

## Source boundary

The Electron desktop explicitly preserves `ui-settings-mcp` and `host/mcp-servers` in `src/shared/harness-desktop-forks.js`. Its active `vendor/deepseek-harness/packages/bundle/web-app/cordis.patch.yml` includes both packages. Behavior is based on `packages/client/ui-settings-mcp/src/client/McpSection.tsx` and `packages/host/mcp-servers/src/types.ts`, not an unmounted vendor feature.

## Delivered

- List records retain host `writable` and original `spec`; selecting Edit opens the exact existing JSON without dropping unknown fields or changing numeric precision.
- Save binds to the selected id, validates the JSON object/transport/command-or-URL, calls existing `mcpServers/upsert`, and reloads the list. Failure retains draft and shows error; a newer draft survives an in-flight save.
- Remove requires a displayed target and a second confirmation; cancellation sends nothing. Queued confirmation remains bound to the displayed id. Failed removal retains target/error; success refreshes and closes a matching editor.
- Composition/read-only rows hide editing, deletion, and enablement actions; controller independently rejects these mutations, including raw JSON overwrites of read-only ids.
- Serialized mutations reject repeated clicks. Errors from toggle, retry, authorization, test and deletion are no longer discarded. A write followed by failed refresh is reported separately from a failed write.
- Production RPCs remain existing APIs and use a 45-second request timeout. The patch was tested only against in-memory fixture data, without credentials, real accounts, permissions changes, or real MCP processes.

## Verification

Focused `go test ./internal/desktop/mcpconfig` passed. The single core fixture covers selection, read-only protection, rename rejection, error retention, repeated save/delete, concurrent newer drafts, unknown-field preservation, cancellation and successful removal/reload. Windows compilation is delegated to the integration task; no full-suite/race pass is claimed. Native runtime UI clicking was not performed in this cloud environment.

The editor remains the existing raw JSON editor; this change closes the missing lifecycle rather than adding a separate form wizard.
