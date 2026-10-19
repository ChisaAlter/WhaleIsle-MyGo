# Original-source parity implementation batch

Branch: `codex/session-runtime-compat-20261019`. Repository: the task-local `WhaleIsle-MyGo` checkout. Original `C:/AI/Deepseek-Harness-Desktop` is read-only. The installed original, `C:/AI/whale-code`, dirty user checkout, user instances and delivered `WhaleIsle-MyGo-local-9232f52` folder/data are untouched.

The latest instruction is source implementation first, then one concentrated validation phase. New startup runs, mock services, package copying and staged deliveries are paused. These commits record implementation, not product acceptance.

| Area | Original source / implemented difference | Source commits |
|---|---|---|
| Launcher/components/import | `src/renderer/launcher-components.js`, `launcher.js`: running controls, per-component pending/status, real management menu, unsupported states, workspace groups and partial selection, keyboard navigation, diagnostic/recovery actions; component list runs outside the input/render thread | `55beab7`, `12ea3db`; shared main wiring pending integration commit |
| Task lifecycle | `src/main/index.js`, `launcher-service.js`: asynchronous Stop/Restart/Close/update/uninstall, original task confirmation retained; live local PTY work is counted and approved runtime changes clean up its resources | `62c20f3`, `600ceaa`, `edba766`, `54a25bc`; main wiring pending |
| Conversation/composer | Original session-controller, input facade and QueueDock: late Stop ownership, convergent steer results, command echo/engaging behavior, canceled splice receipts, failed attachment restoration and stale queue editor retirement | `2b7dca9`, `7d37fea`, `5924f4c` |
| Bot/Room/routine | Original `vendor/dshbot/client/client.js`: cached model/reasoning/default selection at creation, cached Host workspace selection, routine context/silent/watch values submitted together in the original save operation | `f7f49f7`, `ed79b3e`, `3b28db7` |
| MCP/Skills/plugins | Original client modules: explicit failed-managed-connection retry, connection/tool summaries, full skill creation, source buckets and metadata search, scope retirement, preset-provided plugin status | `898c89e`, `23d2617`, `d00faab`; main scope hook pending |
| Presets/install | Original preset descriptions/Markdown guides; install close cancellation, aborting inspection, too-late and lost-response reconciliation of the existing request, without resending install | `59fdb1d`, `0f2485b`, `60512f5` |
| General/About/Usage/Appearance | Original settings CSS/components and usage styles: role-specific font controls, narrow field saves, pending family blur/navigation flush, immediate narrow scheme selection, real Harness version, typography, heatmap/KPI density, responsive layout and loading/error states | `92a9b56`, `412914c`, `98060f3`; shared main wiring pending |
| Project | Original `ProjectTaskCard`, `RequestDiff`, `ProjectTaskDelivery`, `ProjectVisualResults`: missing requirement changes/revert paths, compact Undo, released/settled work projection, human status priority, real page attachment object/height decoding and published-page reader | `1b9ba61` plus root integration work in progress |

Static inspection also prevented duplicate implementations: existing Bot route editing, Room reply recovery, notification suppression and task retry already had product paths. Original Agent Presets has no direct edit/delete API, so none was invented.

Correction to earlier evidence: `9da81d3` fixes only the base update-check callback. `withNativeReleases` overrides the actual launcher callback with an already asynchronous release refresh. The gated helper experiment is not the actual launcher button's root-cause or acceptance evidence. `995cbe0` remains the independently demonstrated exact-line-height correction.

An attempted early `go build -o NUL ./cmd/whaleisle` was rejected by automatic approval review before execution because source work was still incomplete. It generated no compile result or binary. No workaround or alternate compile was run. Source work continues; comprehensive regression, Windows build, native/OS operation and packaging remain deferred until the implementation scope is finished.

Remaining implementation work in this batch is tracked explicitly rather than called acceptance: responsive and interactive Project page presentation, native requirement-diff presentation, and the remaining source-backed remote/IM state gaps. GPU fluid/GIF expansion remains paused by the earlier priority decision. Matching original/Go desktop pixels and physical OS input still require the supported native computer-use capability. Existing older receipts apply only to their recorded source and operation.
