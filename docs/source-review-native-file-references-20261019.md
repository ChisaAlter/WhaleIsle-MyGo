# Native historical file reference source review

Date: 2026-10-19. This is a bounded source review of `3b4567d`, `8281fd9`, `84fd323`, and the source corrections described below. No test, build, Host, browser, screenshot, account, or service was run for this review. Source comparison does not establish runtime acceptance.

## Actual original navigation and reader ownership

The original reference tree is `C:/AI/Deepseek-Harness-Desktop/vendor/deepseek-harness`; all original files were read only.

1. `packages/client/ui-reference/src/client/index.ts:64` injects `openFile` using the reference's originating `sessionId`, resolves its working directory, and awaits `workspaces.openPath(path, {sessionId})`. `FileReference.tsx` contains its own request ticket and failure/retry/close modal. A successful navigation clears that local failure; preview content loading has a separate owner.
2. `packages/client/ui-surfaces/src/client/openpath-intercept.ts:64` performs desktop takeover when listing IPC exists. `apply.ts:198` (`openWorkspaceSurface`) resolves the path relative to that session's cwd, opens `files` for its root, and otherwise calls `sidebarRight.openResourceIn` with a session file address. Paths outside cwd fall through to the base Host system-application opener. Browser-document preview IPC failures after the Files navigation leave Files in place and do not reject it (`apply.ts:165`).
3. Resource navigation alone does not determine the bytes reader. `packages/client/ui-files/src/client/desktop-files.ts:65` accepts session-relative ordinary files with a cwd, declines absolute resource paths and Office formats, and registers the desktop type at `extension` priority (`:125`). `ui-sidebar-documentpreview/src/client/definition.ts` registers the Host text viewer at `fallback` priority.
4. The desktop type uses `ui-files/src/client/shell.ts`'s `window.shell.readFile`, `readFileMedia`, and `writeFile`. `FilePreview.tsx:201` specifies editable complete text, 500 ms autosave, retained dirty buffers, and explicit save coordination. This includes ordinary historical references taken over by that type. The presence of a session address does not convert this surface to a read-only Host preview.
5. The fallback text viewer uses `ui-sidebar-documentpreview/src/client/index.ts:107`'s session-scoped `remote.workspaceFiles.readBytes`. `TextPreview.tsx` / `face.ts` use tab lifetime, cancellation, renderer/read state, and a separate content failure. Its image renderer builds its source from those bytes. This contract must not be applied indiscriminately to the desktop type.

## Findings and corrected conclusions

### P2: waitable reference completion did not request repaint — source corrected

In the reviewed callback chain, `OpenHistoryReferenceOnHost` completes on a worker; main marshals completion through `mygo.RunOnMain`, and the inline reference changes its own failure state. `third_party/mygo/app.go`'s `Dispatch` only drains queued callbacks. `RunOnMain` does not itself invalidate Content; `Window.Update` and `Window.Invalidate` are the explicit repaint APIs. The initial completion could therefore change the failure/retry modal state without requesting a new frame.

`9b72ad6` adds `win.Invalidate()` after `complete(err)` while retaining captured-session/client checks and UI-thread delivery. The same commit shares the captured context readers between reference navigation and the frame. Source reread confirms both changes. This finding is source-closed, not runtime-tested here.

### Withdrawn P1: ordinary workspace text/image must always use Host bytes

The initial review followed the Host fallback viewer without first accounting for desktop resource-type takeover. That produced an incorrect broad conclusion that a session-relative historical reference hitting a local same-named file must be wrong. The original desktop type explicitly reads and edits that local workspace file. Blanket Host-first reading also sets native `FileTab.Remote`, which `ensureSaveQueue` rejects; it would disable the expected desktop save path.

The proposed blanket text change was committed as `6835a95`, with unexecuted fixtures in `be7575b` / `d03cf35`. After reading the actual interceptor, registry priorities, desktop viewer and shell, `5e918af` precisely reverses those three product-file changes and removes the mistaken-contract fixture. At that correction commit the three product files match their pre-`6835a95` content; unrelated editor buffer changes and `9b72ad6` are preserved. None of those fixtures was executed. This withdrawn claim is not a remaining product finding or evidence of a QA failure.

Likewise, `84fd323`'s blanket Host image selection is not justified for the default desktop Files type by the fallback viewer's image contract. The Files owner's `be7a34e` removes that selection and its mistaken-contract fixtures; source reread confirms the native image worker again calls `loadImagePreviewContext`, retaining its generation owner. No claim that all Files images require Host-only bytes remains from this review. That source correction was not runtime-tested here.

## Native seams checked and limits

- `internal/store/session/history_file_reference.go` captures the client and endpoint, checks the exact `TurnFileScope` and context before/after Host calls, and resolves relative paths using the captured session cwd. It returns a waitable navigation error without changing the global action banner. Main's completion checks the captured session, scope and client binding before changing the reference's UI state.
- `internal/ui/chat/user_references.go` contains per-reference request tickets, retry and close state. The callback is delivered on the UI thread. A later file-preview read error does not retroactively turn accepted navigation into an inline navigation failure.
- Native root-relative Office tabs have a reachable existing resource-owner seam: `previewPane` → `paintPreviewToolbar` → `prepareDocumentViewer` → main's `openDocumentViewer` → `webview.DocumentURL`. That URL carries the exact Host/session and a validated relative path; the original registry can select its fallback Office/document renderer. `remoteWorkbookReader` separately reads XLS/XLSX bytes through `ReadSnapshot` with captured session validity.
- A historical path outside cwd currently reaches `OpenHistoryReferenceOnHost` and the Host's system-application opener. It is not an implicit native embedded text preview. This matches the original ordinary `FileReference` fallthrough. The original Project-specific `workingDirectory` option deliberately preserves absolute resource identity; its complete native Project routing was not audited in this bounded review.
- The reviewed Host snapshot reader checks session validity, stat/version/path/size and every byte range, and does not fall back locally after a failed Host byte read. The native image queue gates publication by tab, client/session binding, root/path and generation. Reader selection must still respect which desktop or Host resource surface owns the file.
- `workspace_document_preview.go` also supports desktop browser-preview descriptors and explicit local fallback when a Host source cannot be obtained. It is not uniformly a Host-only surface. This review did not infer a new Host-only contract for that separate path.

No new modal-generation finding was established: pending callbacks already carry their captured current-scope predicate and ticket, and the original React reference also retains local modal state for its mounted occurrence. No additional defensive state machinery was added.
