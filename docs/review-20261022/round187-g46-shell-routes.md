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
