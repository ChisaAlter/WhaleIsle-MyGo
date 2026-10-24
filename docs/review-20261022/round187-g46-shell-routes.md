# G46 — remote shell routes served with native auth (round 187)

Date: 2026-10-25. Closes the round-184 blocker with Option 2 (small): mount
`POST /__remote__/shell/<name>` in front of the daemon's mobile web server, with
the native auth/status semantics, and delegate read-only work to the existing git
gate and a rooted listing.

## 1. What changed

| File | Change |
| --- | --- |
| `internal/desktop/phoneremote/shell_routes.cjs` (new) | Route table, auth, statuses, read-only dispatch, rooted `listDir`, redacted `getConfig`, `mountShellRoutes` (prepends a `request` listener, idempotent). |
| `internal/desktop/phoneremote/git_bridge.go` | Embeds `shell_routes.cjs` and wraps it as `const __mygoShellRoutes = (function(){…})()` in the composed bridge script (single temp file; no second file beside it). |
| `internal/desktop/phoneremote/bridge.cjs` | After the daemon client exists, polls `remote.mobileWebServer` (500 ms, `unref`) and mounts the routes there. Tokens: `config.remoteToken` plus the live pairing offer's `authBootstrap.pairingToken` (decoded from `remote.pairing.url`); nothing logged. |

## 2. Native authority

- `reference/electron/src/main/remote.js:708-730` — auth first (`matchingToken`),
  then `POST /__remote__/shell/<name>` before the harness proxy.
- `remote.js:779-807` — 404 unknown name, **501 `shell unavailable` without
  invokeShell**, 413 body >256 KiB, 400 invalid JSON, else `{ok,result}`.
- `shared/remote-auth.js:45-64` — `Authorization: Bearer <token>` or cookie
  `dsh_remote=<token>`; anonymous non-HTML → plain `401 unauthorized`.
- `remote-shell.js:1-109` — the 15-name table; MyGo serves the read-only subset
  and answers the mutation subset with the native 501 shape.

## 3. Route coverage

| Route | Native | MyGo now | Notes |
| --- | --- | --- | --- |
| gitStatus | invokeDesktopShell → git.gitStatus | **served** via `gitGate.git` | live 200 (see §4) |
| gitDiff | git.gitDiff(cwd, options) | **served** | live covered by unit table |
| gitBranchList | git.gitBranchList | **served** | |
| gitFetchForStatus | git.gitFetchForStatus | **served** | cheap pass-through |
| listDir | fs.listDir(cwd, rel) | **served** (rooted: only inside `remoteGitRoots`, no `..` escape, 512-entry cap) | live 200 |
| getConfig | host.getConfig() | **served**, redacted (enabled/mode/port/bind/lanTls/gitEnabled only) | live 200, no tokens |
| gitCommit/Push/Pull/Switch/Create/CreateCR/saveConfig/open* | gated desktop services | **501 `{"ok":false,"error":"shell unavailable"}`** | TODO(G46): wire through the git gate once the desktop prompt is proven end to end; auth never bypassed |

Statuses: 401 anonymous/bad token (plain `unauthorized`), 400 invalid JSON,
413 >262 144 B, 404 unknown name, 200 read-only, 501 mutations.

## 4. Tests and live run

Node request table (`internal/desktop/phoneremote/shell_routes.test.cjs`,
`node --test`, 7/7 pass): 401 for anonymous/Bearer-wrong/cookie-wrong and the git
gate is not reached; 200 for gitStatus/gitDiff/gitBranchList/gitFetchForStatus via
Bearer **and** cookie; listDir inside the root ok, root outside/`../..` refused;
getConfig contains no token; 404/400/413; every mutation 501; non-POST and foreign
paths fall through; `mountShellRoutes` keeps the SPA listener and is idempotent.

Go wrapper (`shell_routes_test.go`): the composed bridge script embeds the module
exactly once (and still carries the original git sources), and `node --test` runs
the table with the bundled Node (skips only when Node is absent).

Live probe (`tmp/probe-shell`, temp LAN runtime, temp repo on `probe-branch`,
pairing token decoded from the daemon offer, sha256 only in evidence):

| Request | Port | Result |
| --- | --- | --- |
| POST gitStatus anonymous | 3180 | **401 unauthorized** |
| POST gitStatus Bearer | 3180 | **200** `{"ok":true,"result":{"refName":"probe-branch","hasWorkingTreeChanges":true,"workingTree":{"files":[{"path":"probe.txt",…` |
| POST listDir Bearer | 3180 | **200** rooted entries |
| POST getConfig Bearer | 3180 | **200** redacted config |
| POST gitCommit Bearer | 3180 | **501 shell unavailable** |
| POST gitStatus (no desktop authorizer) | 3180 | 400 `Mobile Git desktop workspace permission expired` — the gate denies, never bypasses |

The mobile server is **3180** (3499 remains the harness-facing Express app and
answers 404 — unchanged). The live 200 requires a desktop authoriser: the probe
installs `SetGitAuthorizer` as the user's "allow this workspace"; in the app the
bridge emits `git-authorize` to the desktop. The pairing token is used in memory
only; evidence stores `tokenHash` (sha256), never the token.

Evidence: `docs/review-20261022/evidence/remote-shell-probe.json`.

## 5. Residual

- Mutations intentionally 501 until the desktop prompt path is proven end to end
  (documented TODO in the switch); no partial mutation wiring.
- `listDir` is a rooted, name/type-only listing (no sizes, no symlink targets);
  the phone UI's richer fields would need a follow-up.
- Relay mode and real-phone pairing still unverified in this slice.

## 6. Round 194 addendum — gated git mutations

`gitCommit`, `gitSwitchBranch` and `gitCreateBranch` are now served through the
existing desktop git gate (never bypassed). `gitPush`, `gitPull`,
`gitCreateChangeRequest`, `saveConfig` and `open*` remain **501 shell unavailable**
with `TODO(G46)`: their desktop services are not proven end to end yet.

### Route status

| Route | Status | Notes |
| --- | --- | --- |
| gitCommit | **served (gated)** | args: `cwd` (string, must be inside `remoteGitRoots`), `message` (non-empty, ≤64 KiB), optional `filePaths` (string[]), `options` passed through; calls `git.gitCommit(cwd, message, filePaths, undefined, options)` |
| gitSwitchBranch | **served (gated)** | args: `cwd` in roots, `ref` (non-empty ≤512) |
| gitCreateBranch | **served (gated)** | args: `cwd` in roots, `name` (non-empty ≤512) |
| gitPush / gitPull / gitCreateChangeRequest / saveConfig / openSettings / openGallery | 501 + TODO(G46) | reasons recorded in the switch comment |

Status mapping: 401 anonymous/bad token, **403** for a gate denial (native error
text, e.g. `Mobile Git is not authorized for this workspace` / `Mobile Git desktop
workspace permission expired`) or a `cwd` outside the configured roots, 400 for
invalid arguments (`invalid args: message|filePaths|ref|name`) and invalid JSON,
413 >262 144 B, 404 unknown name, 200 `{ok:true,result}`. Git results with
`{ok:false,message}` now follow native `wrapResult` (remote-shell.js:39-45): they
become `{ok:false,error:message}` instead of being nested in a success.

### Tests

`shell_routes.test.cjs` now has 10 passing cases, adding: mutations without a
desktop authorisation → 403 with the native error text and the gate still
consulted; bad args (`message`, `filePaths`, `ref`, `name`) → 400 and the gate
never reached; a `cwd` outside the roots → 403; success through a fake authorizer
→ 200 with `gitCommit(cwd,message,filePaths,undefined,options)` argument capture,
plus create/switch results. Go wrapper test unchanged and still green.

### Live probe (temp repo, temp LAN runtime, Bearer token, sha256 only in evidence)

| Request | Result |
| --- | --- |
| POST gitCreateBranch `{cwd, name:"g46-branch"}` | 200 `{"ok":true,"result":{"ok":true,…}}` |
| POST gitCommit `{cwd, message:"probe"}` | 200 `{"ok":true,"result":{"ok":true,"status":"created","commitSha":"3c02b10…","subject":"probe"}}` |
| POST gitCommit again (nothing staged) | 200 `{"ok":true,"status":"skipped"}` |
| POST gitSwitchBranch `{cwd, ref:"g46-branch"}` | 200 `{"ok":true,"refName":"g46-branch"}` |
| POST gitCommit without `message` | 400 `invalid args: message` |
| POST gitCreateBranch without `name` | 400 `invalid args: name` |

Post-probe repo state: `branches: * g46-branch, probe-branch`, `head: g46-branch`,
`log: 3c02b10 probe`, `status:` clean, `probe.txt` tracked. The temp repo is deleted
at the end of the probe (reset); the earlier repo had no commits and now carries the
probe commit only inside that throwaway directory.

### Residual

- push/pull/PR/saveConfig/open* remain 501 + TODO; they need desktop services
  (remote/PR generators) proven end to end.
- Gate denial was exercised by the node table (403) and by round 187's live run
  without an authoriser (400 then; now mapped to 403); the live mutation run used
  the probe's `SetGitAuthorizer` as the desktop "allow".
- Relay mode and real-phone pairing still unverified.

## 7. Round 202 addendum — gated gitPush / gitPull

`gitPush` and `gitPull` are now served through the desktop git gate (native
remote-shell.js:70-73 delegates both with only the workspace cwd; the branch is
HEAD's). `gitCreateChangeRequest`, `saveConfig`, `openSettings`, `openGallery`
remain **501 shell unavailable** with TODO(G46).

### Route status

| Route | Status | Notes |
| --- | --- | --- |
| gitPush | **served (gated)** | `cwd` must be a non-empty string inside `remoteGitRoots`; optional `remote`/`branch` validated when present (forward compatibility) but not needed — native passes only cwd. Returns the gate's push result (`status: pushed`, `branch`, `upstreamBranch`, `commitSha`) |
| gitPull | **served (gated)** | same validation; returns `status: pulled`, `refName`, `upstreamRef`; merge/conflict information comes from the git module's result |
| gitCreateChangeRequest / saveConfig / openSettings / openGallery | 501 + TODO(G46) | desktop PR/host services not proven end to end |

Status mapping unchanged: 401 anonymous/bad token, 403 gate denial or cwd outside
roots (`Mobile Git is not authorized for this workspace`), 400 invalid args
(`invalid args: remote|branch|message|…`) and git-level `{ok:false,message}` →
`{ok:false,error:message}`, 413 >262 144 B, 404 unknown, 200 `{ok:true,result}`.

### Tests

`shell_routes.test.cjs` is now 13/13: push/pull without a desktop authorisation →
403 with the gate consulted; outside-root cwd → 403; `remote:42` / `branch:""` →
400 before the gate; success via a fake authorizer → 200 with the exact gate
arguments captured (`gitPush(cwd)`, `gitPull(cwd)`, i.e. native's single-cwd
delegation); a git `{ok:false}` push result → 400 native shape. Go wrapper test
unchanged and green.

### Live probe (temp repo + temp bare remote, Bearer token, sha256 only)

| Step | Result |
| --- | --- |
| POST gitPush `{cwd}` | 200 `{"ok":true,"result":{"ok":true,"status":"pushed","branch":"g46-branch","upstreamBranch":"origin/g46-branch","commitSha":"6964bd1…"}}` |
| bare remote after push | branch `g46-branch`, head `6964bd1…`, log `6964bd1 probe`, files `probe.txt` |
| second clone commits `second.txt` ("second commit" `ef0fd64…`) and pushes | bare head `ef0fd64…`, files `probe.txt`+`second.txt` |
| POST gitPull `{cwd}` | 200 `{"ok":true,"result":{"ok":true,"status":"pulled","refName":"g46-branch","upstreamRef":"origin/g46-branch"}}` |
| local repo after pull | log `ef0fd64 second commit` / `6964bd1 probe`, clean status, HEAD `g46-branch` |

Temp repo, bare remote and the second clone are deleted at the end of the probe
(reset); nothing outside `%TEMP%` was touched.

### Residual

- `gitCreateChangeRequest` still 501: the PR generator needs the desktop review
  service (`git-generate-draft`/`git-review-request`) proven end to end.
- `saveConfig`/`open*` still 501 (host RPC surface not exposed).
- Gate denial: 403 proven in the node table; the live push/pull run used the
  probe's `SetGitAuthorizer` as the desktop "allow".
- Relay mode and real-phone pairing still unverified.
