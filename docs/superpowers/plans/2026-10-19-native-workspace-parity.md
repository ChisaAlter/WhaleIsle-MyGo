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

## Second batch: high-frequency pages outside the workspace

Baseline `0b85f86` (product `ce4efcb`). Preserve the existing ce4efcb EXE and use a separate filename for the next candidate. The original source checkout remains e353e8e13f, read-only. Inventory actual registrations and desktop overrides before changing a page; the detailed difference list lives in the existing status document.

This batch restores the settings shell, curated desktop market/inventory presentation, schedule catalog/detail, workspace selection/rename/remove, and Host directory browser. It does not claim all settings bodies, the separate Plugins manager, Jobs observation, or Projects are restored.

Ownership and shared contracts:

- Settings worker: settings shell/modal/navigation only. Optional `Model.ActiveSection` + `SectionContent` compose a feature-owned body; existing callbacks and dedicated bodies retain their owners.
- Plugins worker: `internal/ui/market/**` and settings `plugin_inventory.go` plus dedicated tests. Optional `market.Model.Embedded` omits the standalone header inside settings. Keep the curated catalog backend; do not impersonate the missing generic Plugins manager.
- Schedule worker: active `internal/ui/schedule/Panel` presentation and Records rendering only. Root removes the duplicate frame task heading and passes the existing close callback. Preserve rule/CAS/history ownership.
- Workspace worker: `internal/ui/workspacepicker/**` and `internal/ui/dirpick/**`. Picker accepts the current `ui.Element` anchor and `OnBrowse`; root publishes a stable Hero button handle through frame/composer. Sidebar Add opens the directory browser. Management stays in bounded modals.
- Directory lifecycle worker: new cmd helper and focused tests only; root owns main.go fields/callsite removal. Immediate open/loading, captured Host client/generation, cancellation and request fences prevent late results reopening or retargeting a closed browser. The directory body accepts Loading/Busy and fills the bounded parent with a scrolling list and pinned footer.
- Root: all cmd/main/frame/composer integration, combined fixtures, review, build and evidence. Shared files have one editor; workers add regressions only after their product changes.

Validation: isolated page fixtures and layout/interaction regressions, then the complete existing main-workspace gate for this batch. Independent Standards and Spec review precedes commits. Build a new Windows EXE from a clean commit; do not replace or restart either user's existing instance. Original native screenshot absence is recorded, not a prerequisite for source-based implementation.

Batch checkpoint: implemented and reviewed in `e565dcd`; all twelve page/workspace test packages passed, followed by the affected external-navigation regressions. Twenty software PNGs and one inspected hidden Windows native settings capture are retained. The separate e565dcd EXE embeds a clean source revision; the ce4efcb binary hash is unchanged. Remaining settings bodies, independent Plugins manager, Jobs, Projects and full directory browser features remain explicitly listed in the status inventory.

## Third batch: complete the fixed remaining page scope

Continue from 0376ce3 through settings bodies, Jobs, Projects, the full Host directory browser and the independent Plugins manager. The integrator retains main.go/lifecycle/composition and makes local commits only after affected checks and independent review. Preserve both earlier candidate EXEs.

Parallel file ownership: settings General/Interface/Appearance/Pet/About; runtime settings Account/Models/MCP/Skills/Agent presets/Remote/Usage/IM; independent pluginmanager controller and UI plus namespace configuration body; Jobs controller/session/header; Projects UI/projectflow; directory browser UI/cmd owner. Shared interfaces are agreed before integration. Root owns cross-namespace described-field writes, settings composition, mainState additions and all remaining cmd integration.

Settings use current Host-described field types and revisions; one-field writes do not select another namespace or clear unrelated drafts. Remote, Remote Workspace and usage statistics remain bodies inside the original modal. The legacy relay token field is distinct from chisacode-v2 pairing. Pet's Whale contribution reuses a Host/generation-owned settings controller and does not create sessions or submit model prompts. Plugins is the original profile manager, separate from the curated desktop market; a missing generic market capability must remain explicit.

Each owner implements before adding focused regressions. Fixtures use fake HTTP/WebSocket or pure UI data only; no actual Host mutations or user configuration writes. The final gate includes the twelve prior page/workspace packages plus newly changed controllers/stores/pages. Retain wide/narrow and loading/error/empty/data evidence. Native self-capture uses an isolated hidden process. Current original-window capture remains unavailable; software/source comparisons do not certify full visual parity.

Third-batch completion: all fixed page/controller work implemented; final five-package gate passed after last review corrections, with 22 affected packages covered across final gates. Standards and Spec report no remaining P1/P2. 247 synthetic page PNGs, 41 workspace PNGs and inspected hidden native Account/Plugins captures retained. Same-state original native capture, real Host mutation acceptance, installed candidate and Windows IME remain open; no live process was touched. Clean candidate provenance follows in the status report.

Windows follow-up disposition:65 component acceptance rows now map original rules to Go, concrete linked regressions and evidence. Source-proven missing session time/pin and floating edit status were corrected; isolated startup no longer mutates installed URL/login registrations, SDK paths, shared hook or lock. Product4004010 plus exact original vector dcfb14c have affected-package checks; dcfb14c clean EXE has actual isolated hidden startup/owned-PID cleanup evidence. No22-package rerun. Remaining source gaps (installer brand/languages, rich curated README, exact fluid renderer) and tool/runtime/installation blockers remain explicitly open in the status report; no complete-original acceptance or publication sign-off. Parent owns the existing30-minute schedule; no duplicate created.

## Source-gap follow-up after88838f0

Implemented in parallel with one integrator: Market rich README and native image cache/UI were feature-owned; shared Markdown document layout, URL activation and main wiring were integrator-owned; installer branding/language pages were independently owned. Fluid fidelity was assessed read-only in a separate stream, with no SDK or animation edits.

README now follows original native document metrics, CSS margin/list/rule rules, explicit inline link behavior, literal HTML and absolute remote-image sizing/alt behavior. Installer restores the original observed resources, welcome/license/finish/uninstall page sequence and Chinese/English language order. Spec and Standards findings were corrected and their bounded delta reviews closed all five P2 reports. Final Markdown/Market gates passed1.866s/2.486s; affected prior Files/Chat/cmd checks and the installer static resource contract passed. See the canonical status document for component-to-test links and PNG evidence.

Exact fluid solver/animation equivalence remains implementation work documented in evidence/windows-fluid-assessment.json; source1440 dye field performance is unmeasured. README complex SVG/animated media and original h3 cascade remain limited. Actual NSIS compilation, installation/upgrade/uninstall, matching isolated Host, native same-state comparison and Windows IME remain unverified. No system install, extra permission, CIM retry, user-process change, CI, push or merge occurred.

## Solver and animation batch after3f643ef

Goal: port implementable original equations and state rules without moving expensive work onto the native UI thread or silently claiming reduced-resolution output is equivalent. Baseline is product519d31b; original source files remain read-only. No CI has been triggered; historical CI failure count remains unknown, so none is authorized here.

Source contract: cursor-fluid.ts uses aspect-correct short-edge128 velocity/pressure/curl/divergence, separate1440 RGB dye, additive aspect-correct splats (radius.002 at100%,force6000*speed/100,color.15; click10x), curl/vorticity3/divergence, pressure decay.1,20 Jacobi passes, pressure-gradient subtraction, bilinear advection, rational velocity2/dye3.5*speed dissipation,dt<=.016666, normal shading and alpha=maxRGB. Resize preserves fields; updates retune live; hidden/dispose stop scheduling; idle parks after4s.

Options: existing Painter.Effect is stateless/one pass and cannot host persistent ping-pong fields. Extending D3D11 for source-sized half-float textures also needs GL/Metal/CPU ownership and disposal contracts and real GPU profiling; no shader/tool installation is permitted. A pure CPU solver with current Painter.Image can preserve equations and use an event-driven background job, but full1440 dye has a large unavoidable allocation/work cost. Implement the solver first with explicit dimensions, measure original128/1440 against bounded native dimensions and baseline, then choose integration only if measured cost is acceptable. Keep full-resolution/default-path limits explicit; no automatic silent fallback or new setting.

Ownership/interfaces:
- Solver worker owns new third_party/mygo/ui/fluid_solver*.go only. package-ui unexported splashSolver: newSplashSolver(width,height,simResolution,dyeResolution int), Resize(width,height int), Splat(x,y,dx,dy float32,color [3]float32,size float32), Step(dt,speed float32), Render(dst *image.RGBA,shading bool), ImageSize()(int,int), MemoryBytes()int. Coordinates/deltas are normalized bottom-left; Splat receives already force-scaled velocity and intensity-scaled color; Size is100-based. No goroutines or renderer wiring here.
- Integrator owns decorative_effects.go, engine lifecycle, bounded event-driven off-main execution, shared state, final profile decision, canonical mapping and Windows build. At most one in-flight job and one coalesced request; never wait for a solver on UI thread; no idle timer/goroutine spin; dispose releases pending work. Target UI handoff below0.5ms, steady worker below4ms/frame and retained native buffers below4MiB at1440x900. These are acceptance budgets to measure, not predeclared successes.
- README worker owns market/readme.go and dedicated README tests; prove original CSS loading/cascade and correct h3 locally without changing Chat. Audit decoder/original dependency support read-only; do not add libraries or fake media parity.
- Animation worker owns composer/typing_effects.go, beam.go and dedicated trajectory tests only; source tokens/easing, activation clocks and exact cheaply computable state rules. No solver/SDK overlap, no new product controls.

Order: [x] Implement numerical solver and bounded animation/README fixes; [x] then add fixed-seed/invariant/resize/dispose/scheduling regressions and timing/allocation measurements; [x] integrate only the measured safe rendering path; [x] bounded independent delta review and affected checks; [x] clean local commit and Windows build. Numerical tests must include quiescent preservation, finite values, projection/divergence, rational dissipation, clamp and image premultiplication; scheduling tests must prove no queues or wakeups persist after disable/close/idle. Synthetic evidence cannot certify native appearance, GPU duration, Windows IME or source-window parity.

Measured decision: bounded64/64 source-equation CPU worker, at most60Hz, one in-flight task, retained native bitmap. Original128/1440 CPU measured1.458s and93.7MB and is excluded from default. Warmed native worker354 frames: mean1.561ms,p95=2.509ms,max3.062ms,0alloc; UI copy548.2ns; field/output/UI data417792B at1440x900. Old approximation mean.379ms/4alloc: active total CPU is higher and no desktop GPU/CPU equivalence is claimed. Final SDK gate1.115s and Composer/Markdown/Market2.486/1.162/1.582s passed. Native GPU resolution/precision and remaining beam/media pixels remain explicitly open in canonical status.

Final product bf347ff includes physical-DPI shading correction (targetPASS1.261s), clean Windows EXE built and preserved separately. User has now explicitly authorized normal push of this existing isolated branch to the verified private origin; force push/main merge/installation/CI remain prohibited. Actual publication outcome must be verified with remote HEAD and reported separately.
