# Native source integration, 2026-10-19

This records the source-first continuation after `ffe7ab6`. It is not a new test,
build, screenshot, device, or full-product acceptance receipt.

- Checkout: `C:\Users\48818\Documents\Codex\2026-10-19\task-2\WhaleIsle-MyGo`.
- Branch: `codex/session-runtime-compat-20261019`; no merge, push, release, or CI.
- Reference: original repository commit `e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`.
  Its worktree contains pre-existing modifications. Reference behavior is read
  from `git show e353e8e:<path>` or from an explicitly checked clean file.
  Neither the original repository nor the user's other checkout was modified.

## Integration responsibilities

Six exclusive source owners implement conversation/voice, workspace resources,
settings, remote/Pet, extensions, and effects/media. Root owns main/frame and
their cross-domain contracts. Source implementation precedes preparing new
regressions and coordinated validation, as the user explicitly requested.

### Conversation, voice and media entrypoints

- A selected session with a live `Presentation` has active conversation content,
  even before text messages arrive; it does not show the welcome hero.
- Slash-menu selections call the captured session/Host command executor.
- Voice settings navigate to the actual optional voice bundle in the Plugins
  manager. The manager renders the provider/language/preparation/microphone page;
  hiding the page or window relinquishes device preview ownership.
- Speech catalog watching reconnects with the real event transport ready path.
- The composer card uses the reference 22 DIP radius. Beam draws below children,
  with the hairline above; its inner wash cannot cover text.
- SVG owners carry window, selected session and Host epoch into actual Composer,
  Files and Market callers. The existing preview helper uses inert image mode;
  inline SVG routes use `BuildSVGImage`, not an interactive HTML document.
- Markdown respects explicit secondary font sizes; primary chat callers use the
  accepted content preference rather than an unconditional 14 DIP override.

### Files, Browser and terminal ownership

- Browser workspace membership comes from authoritative direct workspace rows,
  with loading/error readiness and session fallback. No parent/CWD inference or
  config fallback can join unrelated partitions.
- Every Files resource pane owns its remote tree, tabs and drafts persistently.
  The remote renderer receives that pane's actual `session.Store`; hidden pane
  retirement cancels transient requests without erasing drafts. An unresolved
  remote mirror still permits the existing local Files renderer.
- Closing a resource pane checks local and remote drafts. Save-and-close first
  preflights both domains, queues explicit writes once and waits for their
  receipts. Errors, changed owners and newer drafts keep the pane present.
- Host terminal descriptors persist identity only. Restoring adopts a matching
  existing terminal; explicit new terminal tabs carry a separate runtime fresh
  marker. Clones/restored descriptors never allocate a shell implicitly.
- Local terminal resize flushes before hiding the drawer/window or changing its
  workspace, in addition to the terminal's own disposal flush.
- Dock splits are horizontal, at most two docked seats, with .2 minimum fraction
  and a measured 100 DIP chip-fit condition. Center/tab moves do not consume a
  new split. Malformed IDs/trees are rejected before content restoration.
- Valid older Go vertical or larger layouts migrate without dropping resources:
  splits become horizontal and surplus seats fold into retained tab groups.
  Drafts and floating resource identities remain attached to those resources.

### Project and desktop source findings

`PROJ-01` deeper comparison follows pinned `vendor/dsh-project/client/client.js`
`SidebarPage`, `ProjectRow`, `ProjectHeader`, `command` and `refresh`, plus the
native project bridge and downstream create service:

- Project title filtering uses the shared sidebar query, not a second input.
- The row overflow contains Rename, Open Directory, Archive/Restore. Detailed
  Project routes remain accessible through the original observer entrypoints.
- Empty active Projects expose the reference Add action. Selected folder tint
  follows selected foreground; the coordinator header opens its directory.
- A failed list refresh preserves prior projects. A confirmed rename/create
  receipt remains accepted even when the subsequent read fails; mutation is not
  replayed. Rename uses the original 240 UTF-16-unit bound.
- Create already delegates to the original Host `select-directory` bridge. An
  additional native picker would prompt twice and was not added.

`DESK-03` normal window defaults are 1440x920 and minimum 960x640. A second
instance with `--background` does not foreground the active instance; initial
background startup shows inactive when not configured to remain hidden. The
reference 32x32 caption buttons with top/right 8 DIP are an independent overlay,
not another 48 DIP row in the content layout. SDK caption/rounded-frame source
work remains separately recorded by the effects owner.

`DESK-04/05` deeper physical chord recording, stale application menu callbacks and
fixed-reference desktop integration are assigned to the shortcuts source owner.
No new deeplink routes are inferred: the fixed original has no desktop custom
URI registration matching the existing Go extension.

### Settings, Plugins and Git shared wiring

- Accepted Host `ui-theme` is the connected application authority. Local config
  is the launcher/pre-Host source; ordinary Desktop Save does not mirror Host
  scheme or image data into launcher config. Pending Host writes preserve the
  native editor draft. Wallpaper data URLs decode in memory.
- The account introduction is the original resumable desktop onboarding flow,
  using display-safe account/balance and credential metadata. It sends no model
  request and does not read a credential literal.
- Subagent and web settings use their specialized served forms once, without
  rendering a duplicate generic namespace form below them.
- The desktop curated Market is always available as in the original; unrelated
  disabled plugin preferences remain preserved.
- Plugins Creator uses its actual `marketOpen && marketManager` page owner and
  the existing captured `CreatorTask` admission: create one session, apply
  `cordis`, then open. It does not seed or send a prompt. Unknown outcomes retain
  the exact target with Locate/Open recovery actions.
- Install registry choice persists as a non-secret native preference, with load
  and save errors returned to the manager. No registry action was executed.
- Git titlebar consumes the existing snapshot's local `RepositoryStatus`, shows
  ahead/behind and uses source `CanPush`/`CanCreatePR` gates. No extra fetch,
  account lookup or PR creation is performed by status rendering.
- Assistant configuration failures reach the Pet settings error receipt.

## Evidence status

Source-owner commits include `67a6924`, `590e30f`, `b103ae7`, `b2d7273`,
`bcf3d5f`, `5c2e6fc`, `4bfd992`, `ff6ac28` and `91e7916`. Domain records retain
the exact branch/state evidence and explicit unreviewed branches.

Root has formatted its exclusive source modules and checked their whitespace.
Shared `main.go`/`frame.go` were edited with narrow hunks, preserving concurrent
work. No new-phase tests, fixtures, builds, shader compilation, app/helper
startup, device acquisition, Host mutation or user configuration operation was
performed for this source integration milestone.

The previous phase's passing component/build receipts belong to its older
source revision. They do not validate these new changes. Real Windows native
visual comparison, GPU/helper containment, full external-service behavior and
packaging remain separate acceptance obligations, not inferred successes.
