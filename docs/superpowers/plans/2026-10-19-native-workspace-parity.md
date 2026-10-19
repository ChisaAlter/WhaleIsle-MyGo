# Native workspace parity implementation plan

Goal: restore the original WhaleIsle main workspace in the native MyGo UI, using the original as the visual authority.

Architecture: existing native frame, sidebar, composer and session store remain responsible for their current regions. The integrator owns cross-module wiring. No new product entry points or replacement design.

## Baseline and constraints

- Local source `C:\AI\WhaleIsle-MyGo`, main `b7fb9f9`; running PID 24668 uses the root `whaleisle.exe`, embedded Go build identifies b7fb9f9 + dirty, built 2026-10-19 08:53:05 local. Preserve the three uncommitted crash fixes in main.go, quickchat_view.go and resize.go; original checkout stays untouched.
- Read-only original: `C:\AI\Deepseek-Harness-Desktop` at e353e8e13f; running original `C:\软件\Whale Isle\Whale Isle.exe`. Do not assume that executable matches the source revision.
- Dedicated isolated copy and branch `codex/native-workspace-parity-20261019`; no main edits, PR #2 merge, push, release, model requests, credential access, session deletion, config reset or security/network changes.
- User explicitly authorizes planning followed by parallel implementation. Implementation precedes new regression tests/fixtures, per execution-constraints skill.
- Existing user-instance screenshots unavailable in this execution surface: computer-use needs node_repl, which is absent; CUA native APIs disabled. The project's Window.CapturePage facility successfully captured an isolated native fixture at1200x800 DIP /1800x1200px (150%). Source JSX/CSS remains the primary implementation specification; same-state original-instance visual comparison remains open.

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

- Layout worker: `internal/ui/frame/**` except `workspace_panes.go` and its dedicated tests after the inspection handoff. Own phase rendering and shared status placement. Consume `session.ViewState.HistoryLoading`, `HistoryReady`, `HistoryError`, `SelectedSummaryBlank`, `PromptAttempted` from state worker. Host-proven blank may show Hero before loading completes; unknown history settles. Composer stays mounted but invisible and noninteractive while settling. Unify Hero/active card-width calculation; composer owns its 16 DIP side clearances.
- Sidebar worker: `internal/ui/sidebar/**` only. Replace selected action strip with original-style overflow menu; retain callback contracts and DnD.
- Composer worker: `internal/ui/composer/**` only. Keep Model signature compatible; preserve copied resize crash fix. Recover original compact tool disclosure/voice controls/height behavior without deleting features or clearing preferences.
- Theme worker: `internal/ui/theme/**` only. Confirm original font stack and control tokens, preserve explicit overrides.
- State worker: `internal/store/session/**` only. Add snapshot booleans `HistoryLoading`, `HistoryReady` and string `HistoryError`; follow selection/generation owns state. Authoritative snapshot marks ready; errors remain visible and retry uses existing FollowSelected. Never equate zero messages with completed history or infer loss.
- Integrator: cmd wiring, isolated native launch/build, review, final evidence/status; workers do not commit, mutate original repos or run shared full builds.
- Batched native-renderer inspection found additional core vertical-slice drift: permanent assistant action/feedback diagnostic buttons and permanent directional/docking/reset rows in the right pane. A source specialist owns `internal/ui/chat/**` after the shared status helper handoff; another owns only `internal/ui/frame/workspace_panes.go` plus dedicated tests. Keep all existing actions in source-equivalent compact controls/menus; no other file overlaps.

## Execution and checks

- [x] Discover actual source/process/version/dirty baseline and create isolated branch.
- [x] Record acceptance baseline and non-overlapping ownership before edits.
- [x] Implement five workstreams, resolve shared contract changes through integrator; source-backed chat/feedback/right-pane corrections included.
- [x] After implementation add focused regressions: replay-to-empty/history/error/stale selection, Hero-first-message width, sidebar action identities, composer overflow/resize and font override behavior.
- [x] Run targeted native tests, Windows build and isolated non-network UI process where supported. Results in `docs/native-workspace-parity-20261019.md`.
- [ ] Obtain real same-size/DPI/state original/native screenshot comparisons. Original user-window capture is blocked by unavailable native automation; isolated native Go before/after is available, but does not close this acceptance item.
- [x] Independent branch review, fix evidence-backed defects, save verified changes as local commits. Review corrections are bounded to source-proven behavior.

CI: not triggered in this task. Historical failure count unknown; no push or CI dispatch permitted by this scope.
