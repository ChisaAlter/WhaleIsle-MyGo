# dshbot and dsh-im source audit

This note records the source-backed boundary used by the MyGo control-plane
client. The reference tree is read-only; no files under `reference/electron`
are changed by this worktree.

## dsh-im

The vendored package does not contain a Typert remote descriptor and does not
register `listMessages`, `getHistory`, `inbox`, `timeline`, or any `im/*`
endpoint. The code generator therefore must continue to scan only
`vendor/deepseek-harness/packages/**/lib/typert.remote-client.js`; adding the
package to `internal/dsh/api` would invent an API that the plugin does not
serve.

The actual host RPC is per-channel Cordis RPC. The shared contract in
`vendor/dsh-im/plugin-src/host/channels/shared/rpc.mjs` exposes
`connection.status`, `bot.bind-credentials`, `bot.reconnect` (with optional
`sendTest`), `bot.delete`, `bot.workspace.set`, and `bot.preset.set`. The
channel RPC files add provisioning, access-policy, sender, callback-repair,
group-response, and legacy connection endpoints. `api.IMCordisChannels` and
`api.IMCordisEndpointCandidates` mirror those source channel names and method
inventories, including WhatsApp.

Because dsh-im has no arbitrary-text send RPC, `internal/store/im` keeps
`im/send` as a probe for a future upstream plugin, but Cordis-only mode records
an explicit local-queue diagnostic instead of pretending that `sendTest` sent
the user's draft. `pluginrpc.Client.IMBotReconnect` exposes the source-backed
`bot.reconnect` plus its fixed `sendTest` option for connection checks. Remote
history remains a local persistence fallback until a future vendor release
supplies a real history contract.

## dshbot and dsh-whale

`vendor/dshbot/lib/control-plane.js` registers the `/dshbot` HTTP control plane
with roster, activity, group-runtime, memory, objective, task, routine, and
catalog commands. It does not register chat or pet endpoints. Chat/pet RPCs are
provided by `vendor/dsh-whale/lib/index.js` under `/dsh-whale` (`pet/state`,
`pet/select-model`, `pet/chat`, and `pet/look`).

`pluginrpc.Client.DshbotDescribeBot` now forwards the optional `botId` used by
the source `describe` command for capability discovery. `DshbotCommand` keeps
the remaining source-defined dshbot command surface available without
silently dropping command-specific fields.

The remaining gaps are source/platform boundaries: dsh-im message history and
arbitrary send cannot be implemented from the current vendor source, and
Windows/macOS/Android runtime acceptance still requires those real platforms.
