# Native workspace parity implementation plan

Goal: restore the original WhaleIsle main workspace in the native MyGo UI, using the original as the visual authority.

Architecture: existing native frame, sidebar, composer and session store remain responsible for their current regions. The integrator owns cross-module wiring. No new product entry points or replacement design.

## Baseline and constraints

- Local source `C:\AI\WhaleIsle-MyGo`, main `b7fb9f9`; running PID 24668 uses the root `whaleisle.exe`, embedded Go build identifies b7fb9f9 + dirty, built 2026-10-19 08:53:05 local. Preserve the three uncommitted crash fixes in main.go, quickchat_view.go and resize.go; original checkout stays untouched.
- Read-only original: `C:\AI\Deepseek-Harness-Desktop` at e353e8e13f; running original `C:\软件\Whale Isle\Whale Isle.exe`. Do not assume that executable matches the source revision.
- Dedicated isolated copy and branch `codex/native-workspace-parity-20261019`; no main edits, PR #2 merge, push, release, model requests, credential access, session deletion, config reset or security/network changes.
- User explicitly authorizes planning followed by parallel implementation. Implementation precedes new regression tests/fixtures, per execution-constraints skill.
- Screenshots unavailable in this execution surface: computer-use needs node_repl, which is absent; CUA native APIs disabled. Visual acceptance remains blocked until actual captures are available.

## State/page difference checklist and acceptance baseline

Compare at equal window client size, DPI, font and data state. Normalize pixels by actual scale; 420px sidebar / 51px send at 150% are 280/34 DIP, not evidence of faulty DPI handling.

1. New blank conversation: original centered Hero; identical card width to active composer, compact tools, no standing attachment/paste/busy-mode action strips.
2. Selected history awaiting replay: original settling stage keeps composer invisible; no welcome flash, stale history or misleading empty state.
3. Active conversation: transcript and composer share conversation-column width contract; first-message transition cannot jump card width. Default sidebar 280 DIP, session rows 32 DIP; selected row must not expand into permanent actions.
4. Failed/disconnected history: visible shared status, retry available, no hidden error under an empty Hero.
5. Sidebar long labels and row menu: one compact row, overflow actions accessible by pointer and keyboard, callbacks retain session identity.
6. Composer: original tool disclosure, input natural height; user-resized height remains a preference, no reset. Narrow widths and saved resize must remain usable.
7. Font/theme: original font fallback and semantic tokens; explicit user overrides preserved.

Verification order: region geometry and position; hierarchy/density/menus/hover; typography/colors/radii/spacing; functional paths. Source tests/build do not certify visual parity.

## Parallel ownership and contracts

- Layout worker: `internal/ui/frame/**` only. Own phase rendering and shared status placement. Consume `session.ViewState.HistoryLoading`, `HistoryReady`, `HistoryError` from state worker. Hero eligible only when not loading/error and actual history is ready (or no selected session). A settling selected session does not mount interactive composer. Unify Hero/active card-width calculation; composer owns its 16 DIP side clearances.
- Sidebar worker: `internal/ui/sidebar/**` only. Replace selected action strip with original-style overflow menu; retain callback contracts and DnD.
- Composer worker: `internal/ui/composer/**` only. Keep Model signature compatible; preserve copied resize crash fix. Recover original compact tool disclosure/voice controls/height behavior without deleting features or clearing preferences.
- Theme worker: `internal/ui/theme/**` only. Confirm original font stack and control tokens, preserve explicit overrides.
- State worker: `internal/store/session/**` only. Add snapshot booleans `HistoryLoading`, `HistoryReady` and string `HistoryError`; follow selection/generation owns state. Authoritative snapshot marks ready; errors remain visible and retry uses existing FollowSelected. Never equate zero messages with completed history or infer loss.
- Integrator: cmd wiring, isolated native launch/build, review, final evidence/status; workers do not commit, mutate original repos or run shared full builds.

## Execution and checks

- [x] Discover actual source/process/version/dirty baseline and create isolated branch.
- [x] Record acceptance baseline and non-overlapping ownership before edits.
- [ ] Implement five workstreams, resolve shared contract changes through integrator.
- [ ] After implementation add focused regressions: replay-to-empty/history/error/stale selection, Hero-first-message width, sidebar action identities, composer overflow/resize and font override behavior.
- [ ] Run targeted native tests, Windows build and isolated non-network UI process where supported.
- [ ] Obtain real same-size/DPI/state screenshot comparisons; record missing pages and states honestly.
- [ ] Independent branch review, fix evidence-backed defects, save verified changes as local commits.

CI: not triggered in this task. Historical failure count unknown; no push or CI dispatch permitted by this scope.
