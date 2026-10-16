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

## Confirmed native PR creation gap

`NativeRemoteScreen.kt:166` builds only `message`, `filePaths` and
`options.featureBranch`. Lines 191/193 include `git-create-change-request` in
stacked actions. `DshViewModel.kt:435-452` forwards the same payload for every
step and retains the remaining steps on failure.

MyGo's `git-gate.cjs` correctly requires `preserveProvided: true` and a nonempty
`title` for creation, except for `generateOnly: true` draft requests. Android's
current payload supplies neither. Consequently commit/push can complete before
the PR step is rejected, and retrying that same payload cannot resolve it.
The Web-only draft extension does not add native Compose controls.

Do not remove the desktop review guard or silently supply a title to make this
pass. The complete fix needs an Android draft review/edit/confirm flow that
sends the existing reviewed payload contract and preserves completed-step retry
semantics. That client lives in the original repository. A separate two-file native patch
against the verified commit has been prepared for review: it pauses before the
PR step, supports generated candidates with explicit Apply, and sends reviewed
title/body only on confirmation. Connection/session/workspace checks protect
review and retry ownership. It has not been applied to or published in the
original repository; no Android artifact has been built.

## Verification and remaining boundary

One focused in-memory Node check called the current MyGo Git gate with the
exact native payload shape. It confirmed rejection with the review-required
error and confirmed that the backend creation method was never called. The
check used a fake backend and authorizer; no Git command, credentials, daemon,
relay, network connection or pairing was involved.

A separate focused static source-contract check for the prepared Kotlin patch
passed. Kotlin/Gradle compilation is unavailable in this environment and has
not been claimed. No broad test run, Android build or live pairing was performed. A real Android
connection to matching Windows/macOS packaged runtimes remains separate device
acceptance; server build success and synthetic fixtures cannot replace it.
