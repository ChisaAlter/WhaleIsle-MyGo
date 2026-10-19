# Extensions and caption regression fixtures — 2026-10-19

Status: **prepared, not executed**. Production source baseline for this fixture phase is `b44371f2`, following the source comparison against original `e353e8e`. Root coordinates execution after every fixture owner closes. This document does not claim compilation, test success, runtime UI acceptance, or installation acceptance.

## Prepared observations

| Fixture | Behavior under observation |
| --- | --- |
| `internal/pluginmanager/registry_depth_test.go` | Explicit JSON `registry:null` survives Inspect and Install; normalized offered registries deduplicate; remembered pnpm default remains nil; source-specific registry stays transient; persistence read/write errors remain visible and dismissible; manual input fences a late public mirror probe; recovery is limited to the specified GitHub failure; an obsolete install RPC cannot clear a replacement request's Busy state. |
| `internal/pluginmanager/controller_test.go` | One Install gesture inspects then installs with accepted registry and disabled default; unknown result remains unknown after inventory refresh but Cancel/Close release the input; More pins the document and reloads the accumulated count when its version changes; missing generic market endpoint stays distinct from curated Market. Older confirmation and refresh-unlock assumptions were replaced with the pinned source behavior. |
| `internal/pluginmanager/discover_depth_test.go` | A retired query cannot replace the current listing or opened detail; partial installation result, logs, final exit and authoritative registry attempts survive failure; restart-required remains a saved result. |
| `internal/ui/pluginmanager/install_depth_test.go` | ANSI replay applies CR, backspace, erase, wide/combining characters, OSC removal and SGR before folding; collapsed output retains head/tail; expansion exposes the middle; copying retains the complete raw transcript. The actual profile plugin card passes its image owner to metadata artwork, with a retired owner rejected before helper startup. |
| `internal/ui/pluginmanager/view_test.go` | The expanded narrow installer displays captured command, cwd, raw combined output and final exit. |
| `internal/ui/market/plugin_artwork_test.go` | Dynamic metadata SVG with no owner retains consumer fallback for data/cache sources; installed projection retains disabled, missing-file and restart states while excluding runtime-only and unidentified entries. |
| `internal/ui/chrome/workspace_trailing_test.go` | Source quick-action cases, full/cozy/compact visible text, HideGit and managed-session gates, independent panel toggles, unavailable terminal, and nonrepo Initialize without a branch badge. Accessible action labels remain present in compact mode. |
| `internal/ui/diffsurface/caption_actions_test.go` | Initialize starts on one captured gesture and stays in that cwd; global review is visible on its current page and shares the existing Workbench across navigation; owner retirement cancels pending preparation and ignores its late result; confirmation/generation rechecks admission; opening review never automatically prepares or executes; clean commit_push uses existing push review; branch picker consumes its asynchronous list, create waits for submit, and retired-root list results are ignored. |
| `internal/pluginmanager/functional_runtime_test.go` | Existing opt-in isolated runtime fixture now follows the single-gesture installer, without an extra confirmation or duplicate request. This opt-in fixture was not run. |

## Execution boundaries

The new caption Git fixtures mutate only `t.TempDir` when eventually executed. Repository setup has no remotes, uses fixed local identity, disables hooks/signing and global/system configuration, and strips inherited Git environment overrides. Caption initialization uses another empty temporary cwd. No fixture invokes a remote push, PR creation, account, model, live market installation, or user profile.

The remaining controller fixtures use in-memory Caller responses. The artwork consumer fixture uses a retired owner to reject startup; the cached remote case installs a transport that fails if any HTTP request escapes. SVG rendering, frame playback, native helper lifecycle and intrinsic dimensions belong to the effects/media fixture owner (`readme_svg_test.go`, `plugin_artwork_svg_test.go` and its native acceptance coverage).

## Unverified boundaries

- No tests, build, app launch, real plugin install/update/remove, real registry probe, live Git remote operation or model request were executed in this phase.
- Headless interaction assertions do not establish native screenshot quality, complete caption geometry, DPI behavior, WebView SVG pixels, focus integration or actual user-profile lifecycle. Root owns shared layout/geometry integration and schedules runtime acceptance.
- These fixtures target the changed source contracts. They do not establish exhaustive equivalence of every original registry URL parsing edge case, log escape sequence, plugin contribution, README media format or generic market server failure combination.
- Source evidence and prior unreviewed boundaries remain in `source-depth-extensions-20261019.md` and `source-depth-titlebar-trailing-20261019.md`.
