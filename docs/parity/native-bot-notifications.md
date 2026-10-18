# Native Bot operating-system notifications

This increment connects the existing per-item `notify` setting to real MyGo OS notifications. It is independent of the IM notification switch and tool permissions. It neither stops Bots nor changes their hidden state. It does not claim that storing `notify` is itself delivery.

## Source contract

Pinned source: [WhaleIsle e9f584be, client.js](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/dshbot/client/client.js), blob `e8ee29c0f8ef21d1dddd01e380faee3f2f58a9d3`:

- Lines 48–51: terminal statuses are completed, failed, interrupted; browser permission request is lazy and once per app session.
- Lines 1887–1890: a personal Bot's display title takes precedence over its name; rooms use their name.
- Lines 2087–2091 and 3390: `items` is the complete catalog. The visible/filter list at 3445–3470 is separate. Notifications consume `items`, so hidden entries and rooms are included, respecting each item's `notify` value. Hiding is not notification opt-out.
- Lines 3517–3543: suppress while focused or unsupported; denied permission does not show. Click focuses and opens the item.
- Lines 3544–3582: first observation per Bot is a baseline; later attention false→true, goal entering blocked, and non-silent terminal lastRun with changed routineId/runId may notify.
- Lines 3603–3607: the 1500 ms activity timer requires `view === "contacts"`, control availability and nonempty item IDs. It has no OS visibility/minimization condition. `view` is the contacts/tasks/routines tab state (3359, 3651–3669), not window visibility. The `document.hidden` listener at 1468–1499 controls avatar animation only.

[profile-ops.js](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/dshbot/lib/profile-ops.js), blob `48ddfba3b6ff29c575ba51e32fc51af3726e1f5c`:

- Lines 206–215 derive attention from unresolved approval/question events.
- Lines 217–237 choose the latest routine run from durable run history and expose routineId, runId, name, status, silent and endedAt.
- Lines 241–287 expose canonical botId/sessionId, activitySeq, attention, goal and lastRun. Activity does not provide a server CAS revision; goal is a summary, not a mutable goal identifier. No activity CAS is invented.
- Lines 763–783 accept explicit IDs, reject over 200, and can return an individual inspection error while other entries succeed.

## Runtime and resource boundary

Only the native Bot page starts this consumer. It stops on page departure, mode/account/Host replacement and actual window close/destruction. Hiding to the tray or minimizing the same still-active Bot page preserves read-only observation and allows unfocused notifications. Other editing controls retain their own hide/minimize retirement hooks. It is not an app-global background service. The current window, authenticated client, endpoint/account, catalog revision, canonical session and local event ticket fence delayed replies and clicks. Observer creation, publication and clicks also check the latest mutually exclusive page flags (launcher, settings, usage, IM, market, schedule, remote, WebView and browser preview), so a same-frame navigation cannot revive an old rendered Bot page. These checks are pure state under the main lock; native window getters remain outside it.

Each round reads the existing `describe` projection, derives all catalog item IDs, then calls existing `bot/activity` once with those exact IDs. Empty catalogs do not issue an implicit all-items activity request. More than 200 catalog items reports the Host's limit rather than silently sampling. No markRead, bot/open, prompt, model, task, answer, stop, routine execution or token RPC is issued.

Rounds are serial, with the next 1500 ms delay starting after the prior result is applied; this avoids the original interval's potential overlap. `Window.Update` posts its callback independently of rendering visibility (bundled MyGo `content.go:166–171`), so a hidden window can acknowledge results without being shown. OS/runtime scheduling can still delay background work; 1500 ms is not a wall-clock delivery guarantee. One shared read slot spans retired/replacement observers and each round has a 10-second context deadline. Existing transport bounds each response to 8 MiB. There is at most one pending native Show/permission submission across these observers, at most one current queued/shown reminder per catalog item, and at most 200 item baselines. Terminal dedup retains a bounded same-timestamp high-water bucket (up to 200 keys per item). No new disk cache is written.

A missing notification flag or status field, malformed identity, duplicate response, canonical mismatch, or decreasing catalog revision does not produce inferred events. A per-item inspection error retires only that item's baseline/click target; other valid items continue, and the page reports partial unavailability. A recovered item begins with a fresh baseline. Foreground, disabled and unsupported observations advance baselines so enabling or blurring does not replay old events. Run identity and timestamp dedup reject repeats after transient empty status.

## Native delivery and click behavior

The adapter uses `mygo.NotificationsSupported`, `NewNotification` and `Show`. MyGo requests OS permission lazily where required and returns denial/failure; denial is retained for the app session. Failures are shown in the native Bot page, and this module never claims that OS acceptance proves a user saw the notification. The first permission wait is bounded to one outstanding native submission; MyGo does not expose cancellation for that OS permission dialog. Retired notifications are closed again when a late Show result returns.

Simultaneous edges for one item are combined into one reminder with fixed event text and the original routine name. It does not include private transcript, approval/question payload, or tool output. Focus suppresses new reminders and drops unsubmitted ones. Shown reminder identities remain until state/scope changes so operating systems that focus before invoking the click callback still navigate correctly.

Clicks only navigate the same already-open native Bot page. They explicitly Show, Restore and Focus its exact window before navigation, rechecking original page/Host/account/window/event ownership after each native callback. A callback that changes ownership aborts navigation. A failed window restoration reports an error rather than invoking hidden management. A represented visible item selects its exact ID; hidden/newly-created items use the independent visibility-management helper `revealAssistantBotVisibility(endpoint, client, botID, sessionID)`. That helper only reveals local management UI and pins an exact target. It neither changes `catalog.hidden` nor creates/restores a session. Missing/stale targets never fall back to a later selected Bot. This increment therefore depends on the visibility-management module being integrated before final application compilation.

## Verification

The narrow dependency fixture uses an in-memory HTTP RoundTripper, pure state transitions, and a blocking fake reader. No live Host, OS notification, account data, models or user computer were used. Initial execution failed before compilation because the default build-cache directory was read-only; the cache was redirected to the existing writable workspace. The first actual focused run passed 7 tests plus 8 rejection subcases. A review found the cold-Bot partial-error issue; the authorized correction run covered only the affected new RPC fixture, passing 4 tests plus 7 rejection subcases, including healthy-Bot continuation and recovered-Bot baseline behavior. No old/race/full suite was run.

Core and RPC fixture tests verify the genuine producer fields, all-catalog hidden/room inclusion, explicit-ID batch, empty/over-limit behavior, strict flags, first baseline, all three events, focus/opt-out/unsupported suppression, stale catalog/activity/canonical cases, run dedup, exact click target data, deletion invalidation and serial cancellation. A separate focused background-lifecycle fixture compiles the actual native binder and tracker against strictly in-memory OS/Host dependencies. It covers hidden/minimized baseline retention, background transition submission, restore-before-selection/pinning, focus suppression, close cancellation and same-callback page/Host/account/window/destruction changes. It does not use or prove real OS notification delivery. The native MyGo adapter is production wiring reviewed against the bundled API; real OS behavior and packaged application delivery have not been exercised. The sole integrator performs the Windows application compile and integration guard checks.
