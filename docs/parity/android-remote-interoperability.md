# Android remote interoperability: source-backed boundary

## Scope

The product targets Windows/macOS desktops and the existing Android remote
client. A Linux remote workspace or an SSH host is not the Android client.
Building server dependencies on Linux does not establish Android interoperability
or desktop release acceptance.

## Implemented in current MyGo source

- `cmd/whaleisle/main.go` passes desktop configuration and the running Harness
  endpoint to `phoneremote.Runtime`. It starts the installed original component
  on explicit refresh and stops it during the desktop lifecycle.
- `internal/desktop/phoneremote/bridge.cjs` instantiates original `DshdRemote`,
  retaining its daemon, relay and offer-v2 authentication protocol. Offers are
  shown only while the daemon listens, relay is connected and bootstrap expiry
  has not elapsed. No replacement Android authentication is introduced.
- `git-gate.cjs` and `cmd/whaleisle/phone_git.go` enforce current desktop-session
  permission and canonical workspace identity for every forwarded Git request.
- `device-actions.cjs` exposes public device metadata and calls original rename,
  unbind and rotation APIs. Unbind stops the daemon and revokes Git before the
  original store operation, then restarts through the original lifecycle.
- `mobile-draft-server.cjs` / `mobile-draft-ui.js` extend the served Web client.
  This JavaScript UI hook alone does not prove native Android has equivalent
  PR/MR draft editing controls.

## Current upstream identity and protocol

GitHub `ChisaAlter/WhaleIsle` main resolved on 2026-10-16 to
`a8c931da37376ce7dc7989e6f30b0db885e06d26`. Targeted reads of the following
files use that exact commit, not the incomplete historical recovery snapshot:

- [Android project and platform description](https://github.com/ChisaAlter/WhaleIsle/blob/a8c931da37376ce7dc7989e6f30b0db885e06d26/mobile/README.md).
- [Native Git controls](https://github.com/ChisaAlter/WhaleIsle/blob/a8c931da37376ce7dc7989e6f30b0db885e06d26/mobile/android/app/src/main/java/ai/deepseek/harness/mobile/ui/NativeRemoteScreen.kt#L158-L211).
- [Native action queue](https://github.com/ChisaAlter/WhaleIsle/blob/a8c931da37376ce7dc7989e6f30b0db885e06d26/mobile/android/app/src/main/java/ai/deepseek/harness/mobile/DshViewModel.kt#L435-L452).
- [Relay client](https://github.com/ChisaAlter/WhaleIsle/blob/a8c931da37376ce7dc7989e6f30b0db885e06d26/mobile/android/protocol/src/main/kotlin/ai/deepseek/harness/mobile/remote/RelayClient.kt).
- [Desktop dispatch](https://github.com/ChisaAlter/WhaleIsle/blob/a8c931da37376ce7dc7989e6f30b0db885e06d26/src/main/dshd-git-dispatch.js).

Android is an existing Kotlin/Compose client, not the browser SPA. Its relay
client sends `dshd.host.rpc.request` with `method`/`payload`, and
`dshd.git.rpc.request` with `action`/`cwd`/`payload`. MyGo retains the original
installed daemon for these wire protocols. The desktop Git dispatch routes
`git-create-change-request` to `gitCreateChangeRequest(cwd, body.input || body)`.

## Native PR compatibility in MyGo (AX202)

The unchanged Android client sends only `message`, `filePaths` and
`options.featureBranch`; it has no native PR title/body editor. MyGo now handles
that missing-reviewed-payload case with a parented **desktop PR review** window.
The phone request stays pending while the desktop user enters title/body or
explicitly applies generated candidate copy, prepares an exact repository/base/
head review, and confirms creation. The existing Go reviewed GitHub executor
rechecks the repository, branch, pushed commit and reviewed payload before creating.
Existing Web requests with `preserveProvided` and a title keep their original path.
A commit message is never treated as PR review approval.

The desktop handler has a 105-second context budget; the private bridge expires
at 110 seconds, before the unchanged Android client's 120-second Git timeout.
Cancel, expiry, runtime replacement, device unbind/listener stop and loss of the
current desktop session's Git grant cancel review. Only one review is active per
runtime. Phone errors retain its already completed commit/push steps and remaining
PR step. Existing PR lookup is performed by the reviewed executor; uncertain
creation outcomes are shown as uncertain rather than automatically retried.

This compatibility path uses the existing reviewed **github.com PR** executor.
GitLab/other MR destinations fail explicitly and remain a separate compatibility
gap. A desktop must be available for confirmation; this is not a new native
Android editor and does not provide phone-only title/body review.

Only WhaleIsle-MyGo is changed. A previously published original-Android development
patch is outside this target and is not counted as implementation or acceptance.
Original main and native Android source remain read-only references here.

## Verification and remaining boundary

One focused check on 2026-10-17 passed the in-memory Node gate fixture and
`go test ./internal/desktop/phoneremote ./cmd/whaleisle -run '^TestGitReview' -count=1`.
It covers missing-handler fail-closed behavior, native review delegation, unchanged
Web reviewed/draft paths, revocation, bounded contexts and runtime cancellation.
The command package compiled with no matching tests. No Git command, real PR,
credentials, daemon, relay or pairing was used by these fixtures.

No original code execution, broad test run, Android build or live pairing was
performed. Real unchanged Android to matching Windows/macOS packaged runtimes,
including review-window timing, cancellation and unknown-outcome handling, remains
separate device acceptance. Server build success and synthetic fixtures cannot
replace it.
