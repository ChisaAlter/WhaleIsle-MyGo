# PET-19: automatic local food refresh and pre-feed rescan

Checked upstream: `ChisaAlter/WhaleIsle@e9f584be6d3b491ce777bb6823f30bc2bbc6f195`,
`src/main/desktop-live2d.js`: `show` owns the 60-second rescan interval, `hide`
stops it, `growthSnapshot` refreshes when opening the panel, and `feedTokens`
awaits refresh before consuming. A failed scan can still feed previously settled
food. The source remains local settled usage, not a remote model/API/account.

## Behavior

MyGo's always-visible card now refreshes when shown/reloaded and once per minute.
The existing native scanner and `food.seen` high-water ledger are reused. First
refresh only seeds history, subsequent observed usage deltas add available food;
no timer consumes food, grants growth/care, wakes the pet or generates messages.
Settlement persistence is necessary to prevent recredit after restart. It does
not introduce timer-based care decay writes or a parallel balance.

A deliberate Feed Token click performs a rescan first, even if the displayed
bowl is empty. The existing per-window single-flight gate rejects duplicate or
parallel requests. After the accepted scan settles, the original atomic feeding
operation consumes at most the next level gap exactly once. A failed scan can
consume only the already-settled saved bowl, explicitly labelled `saved food only`.
Cancelled scans, a changed owner/runtime, rejected late completion or failed
settlement save never consume. Failed feeding saves still change no balance/care.

Automatic refresh has a separate flag so a minute scan does not count as user
interaction or keep postponing the 4-minute nap. It never changes the temporary
sleep state. Existing notification, compact-chat and status routes are retained.

Close, native Hide, disable, page reload or Host/runtime identity changes stop the
minute timer and cancel the in-flight scan. Identity includes the Supervisor and
DSH-client owner plus generation and fixed native Node/data-home paths; no cookie
is stored in the owner key or sent to the page. Client identity is read under
`dshMu`. Reshow begins a fresh timer. All reads/scans remain tied to the native
configured data home, never a page argument.

## Verification boundary

One new synthetic Go `-count=1` fixture passed refresh-only/no-consumption,
explicit refresh-before-consumption, duplicate/stale completion, cancellation,
failed save, and failed-scan saved-food fallback. No real session logs, Host,
model or Bot were read/run. This does not repeat the PET-21 fixture or the full
regression suite. The integration owner owns Windows compilation and GUI checks.
