# G46 — remote shell / git dispatch audit (round 184)

Date: 2026-10-25. Scope: does MyGo's staged `dshd-remote` daemon expose the desktop
gateway's shell/git endpoints, and can the phone run `pwd`/git status through them?

## 1. Native route (authority)

| Piece | Citation | Behaviour |
| --- | --- | --- |
| Route shape | `reference/electron/src/main/remote.js:728-730` | `POST /__remote__/shell/<name>` is intercepted **before** the harness proxy, only for POST. |
| Route table | `remote-shell.js:1-109` | `gitStatus`, `gitFetchForStatus`, `gitDiff`, `gitCommit`, `gitPush`, `gitPull`, `gitBranchList`, `gitSwitchBranch`, `gitCreateBranch`, `gitCreateChangeRequest`, `listDir`, `openSettings`, `openGallery`, `getConfig`, `saveConfig`; unknown names → `{ok:false,error:'not found'}` 404. |
| Handler | `remote.js:779-807` | 404 unknown name, **501 `shell unavailable` when `invokeShell` is not a function**, 413 body >256 KiB, 400 invalid JSON, else `invokeShell(name,payload)` → `{ok,result}`/`{ok:false,error}`. |
| Auth | `shared/remote-auth.js:45-64`, `remote.js:708-726` | `Authorization: Bearer <token>` or cookie `dsh_remote=<token>`; anonymous non-HTML → plain `401 unauthorized`. Tokens = the gateway token + paired device tokens. |
| Desktop wiring | `reference/electron/src/main/index.js:155`; staged `src/main/index.js:160` | the Electron main injects `invokeShell: (name,payload) => invokeDesktopShell({name,payload,git,fs,host})`, i.e. the shell routes are hosted by the **desktop gateway server** (`Remote`), not by the harness. |

## 2. Live probe (throwaway LAN daemon, loopback only)

Program: `tmp/probe-shell/main.go` → `phoneremote.NewRuntime().Start` with
`remoteEnabled/lan/3499/127.0.0.1/lanTls=false`, `remoteGitEnabled=true`,
`remoteGitRoots=[temp repo]`; temp repo on branch `probe-branch`. Waited for
`status=ready` (pairing URL `http://127.0.0.1:3180/#offer=<redacted>`), decoded the
offer's `pairingToken` (sha256 recorded, raw token never persisted) and probed with
`Authorization: Bearer` / `cookie dsh_remote=`.

Evidence: `docs/review-20261022/evidence/remote-shell-probe.json`.

| Probe | Port | Status | Body |
| --- | --- | --- | --- |
| `POST /__remote__/shell/gitStatus` anonymous | 3499 | 404 | Express error page “Cannot POST /__remote__/shell/gitStatus” |
| same, Bearer token | 3499 | 404 | same Express 404 |
| same, cookie | 3499 | 404 | same |
| `POST /__remote__/shell/listDir` Bearer | 3499 | 404 | same |
| `POST /__remote__/shell/writeFile` Bearer | 3499 | 404 | same (unknown-name path never reached) |
| `POST /__remote__/shell/gitStatus` anonymous | 3180 | **405 Method not allowed** | mobile SPA server accepts GET only |
| same, Bearer token | 3180 | 405 | same |

Interpretation: 3499 is the harness-facing Express app (no such route); 3180 is the
mobile SPA server (GET-only). The desktop gateway that owns
`/__remote__/shell/*` is **not listening** in MyGo's daemon stack.

## 3. Why (dispatch comparison)

- `internal/desktop/phoneremote/bridge.cjs:51-63` constructs the original daemon as
  `new DshdRemote({getConfig, git: gitGate.git, getHomeDir, execPath, log, getHarnessOrigin, getSessionCookie})`
  — it passes the gated git backend but **no `invokeShell`, no `fs`, no `host`**,
  so the daemon has nothing that serves the shell routes.
- `gitGate` (`bridge.cjs:36-50`) already implements the desktop side of mobile git:
  `git-authorize` (2.5 s desktop prompt) and `git-generate-draft` for change
  requests, which is exactly the authorisation layer `invokeDesktopShell` needs.
- The staged module does contain the gateway (`src/main/remote.js`, `remote-shell.js`)
  and its own wiring (`src/main/index.js:160`), but that code path belongs to the
  Electron main process; the MyGo daemon runner never constructs it.
- Result: shell names map to nothing; the SPA's git/list calls cannot reach a
  handler. This is not a one-line route addition — it needs a `Remote` gateway
  instance (port, TLS/bind, auth token, mobile web root) plus an `invokeShell`
  implementation backed by `gitGate.git`/fs/host services.

## 4. Verdict and next step

**G46 remains open — blocker documented, no half-wiring.** Two viable paths:

1. **Mount the existing gateway** (preferred): in the daemon runner, construct the
   staged `Remote` server with `invokeShell` delegating to `invokeDesktopShell({git: gitGate.git, fs, host})`,
   sharing the daemon's token/bind/port and mobile web root. Needs a port decision
   (the native gateway owns the LAN port; today the SPA owns 3180) and a test that
   the shell routes answer 401/200 instead of 404/405.
2. **Minimal in-daemon handler** in `bridge.cjs`: implement
   `POST /__remote__/shell/{gitStatus,listDir}` in front of the SPA server with the
   same auth (`remote-auth.js` semantics) and wire `gitStatus` to `gitGate.git`;
   mutations stay gated. Smaller, but duplicates the native route table.

Not attempted here because the port/auth/lifecycle decision is product-level
(which server owns the LAN port in MyGo) — reported rather than guessed.

## 5. Residual

- Relay/real-phone pairing unaffected and still unverified in this slice.
- No product code changed; probe program stays in `tmp/` (not shipped).
