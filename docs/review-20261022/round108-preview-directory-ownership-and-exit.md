# Round108 — Windows private-preview directory ownership and exit

The Windows native backend now keeps a handle-derived identity for its original temporary preview directory and removes it through verified handles after browser exit. Renaming the original and putting a new directory at its old path no longer deletes the replacement. The default executable is pass252. Whole-product layout/UI/function/motion parity remains incomplete.

## Root cause and change

The previous cleanup retained `os.Lstat`'s `FileInfo` and later called `os.SameFile` before `os.RemoveAll(path)`. In this local Go Windows implementation, path-based stats defer obtaining the file ID until comparison. After original A was renamed and B was created at A's old path, both records reopened B and compared equal. A fresh baseline reproduction was recorded in Round107; the replacement-marker regression now passes.

`third_party/mygo/internal/windows/preview_directory.go` uses `CreateFile(OPEN_EXISTING, BACKUP_SEMANTICS|OPEN_REPARSE_POINT)`, checks DIRECTORY and no REPARSE_POINT on the open object, and reads `FILE_ID_INFO` (class18: uint64 volume and128-bit ID). This includes the full ID needed for ReFS rather than its nonunique legacy64-bit index. An original metadata handle shares read/write/delete and stays open through environment/controller startup and the browser lifecycle; it permits normal browser use and rename while preventing ID reuse after the object is removed. Closing the owner invalidates its identity, and repeated cleanup cannot revive a stale ID.

Cleanup acquires a separate original-identity-matching handle requesting DELETE and read-attributes but omitting delete sharing. This pins the verified object for the attempt. It opens an independently verified `os.Root` through `parent.OpenRoot(base)`: direct `os.OpenRoot(path)` uses a Windows share mask that conflicts with DELETE access. It snapshots all child names, removes each through `Root.RemoveAll`'s relative/no-follow traversal, and sets `FileDispositionInfo` with a one-byte BOOLEAN on the same pin for final root deletion. It never re-resolves the original path to remove the root. Any access, identity, sharing or deletion error retains leftovers; there is no path fallback, timer retry, directory scan or later cleanup of a moved object.

The fix uses existing syscall/COM mechanisms, no cgo, no added dependency and no per-call native callback allocation.

## Async lifecycle

`preview_isolation.go` now retains identity through the complete environment callback, including COM queries and exit registration. Effective-directory validation and closed-window checks precede subscribing. It rechecks closure after queries/registration; a window closed before any controller request releases references without requiring a browser-exit event that may never occur.

`webview.go` tracks a pending private controller and whether a returned controller started the browser. An immediate close is reconciled after its completion. Started browsers retain their exit subscription until `BrowserProcessExited`. Browser exit is terminal for this private environment: it stops owned controller retries before directory cleanup and prevents the same environment from recreating an unowned profile at the old path. An unexpected exit ends the preview window; the caller can open a fresh preview.

COM teardown detaches published references before Release, and publishes terminal state before unregistering events. An exit reentered during controller setup waits for the outer callback before tearing down its interfaces. Terminal state also prevents ready publication and further queued calls. Timer cancellation requires a still-live registered HWND, so a late exit cannot cancel a timer on a newly reused handle.

Three independent read-only source reviews covered identity/sharing, async lifetime and COM reentrancy. Their concrete findings were fixed before final acceptance.

## Acceptance evidence

The opt-in real Windows/WebView2 fixture is `third_party/mygo/internal/windows/preview_native_test.go`. It runs all native calls on one locked COM thread, uses its own temporary profile tree and production-shaped local HTML capability, and never starts a Host/model request or opens a user profile. It exercises:

1. Normal private WebView creation, local HTML load, controller close and actual BrowserProcessExited: original profile removed, environment/subscription/identity handle released.
2. Immediate close during startup: on this installed runtime the environment completed synchronously and the controller remained pending. Its completion released identity and subscription; the fixture does not claim a real delayed-environment callback. Separate filesystem/lifecycle regressions cover pending-environment success and failure. Aborted/pre-controller leftovers may be retained conservatively.
3. First returned controller forcibly closed by the existing `TestFailWebViews(1)` fixture, followed by the actual browser-exit event during retry backoff: one failure recorded; original profile removed; environment terminal; stale retry calls did not recreate it.

The first draft fixture used an ordinary URL that the existing local-HTML capability policy correctly denied (navigation status14). It was corrected to the required64-hex-token/document.html route. Later draft assertions assumed all startup callbacks were asynchronous and that Destroy could not synchronously settle a pending controller; those assumptions were removed. Only the final production-shaped, phase-aware run is retained as acceptance evidence.

Final [native/filesystem log](evidence/preview-pass252-native-and-filesystem.txt) passes14 filesystem/lifecycle top-level cases plus the real native fixture; nested cases cover missing/file/directory/unknown-ID replacements and root/child junctions. It also checks1,100 entries, a long Unicode path, pin-time rename refusal, rename availability during browser lifetime, in-use file preservation, terminal repeated cleanup and deferred setup exit. [Verification matrix](evidence/preview-pass252-toolkit-checks.json) records commands, target platform/architecture, exit codes and durations.

## Build and checks

All passing checks used CGO_ENABLED=0; final matrix and executable use Go1.27.2 explicitly:

```text
MYGO_PREVIEW_NATIVE_TEST=1 go test ./internal/windows -run '^TestPreview' -count=1 -v -timeout=100s
go test ./internal/windows -count=1
go build ./...
go vet ./...
go vet -tags mygo_noinspector ./...
GOOS=windows GOARCH=arm64 go build ./...
GOOS=linux GOARCH=amd64 go build ./...
```

The source of the reproducible matrix is `scripts/verify-preview-cleanup.ps1`. The native acceptance and complete Windows package pass. Windows amd64/arm64 and Linux toolkit builds, Windows vet and production-noinspector vet all pass.

The [full toolkit log](evidence/preview-pass252-full-toolkit.txt) has only the two previously established UI failures: `TestDiffFrameAllocs` (steady6/sidebar146/diff148/within-rows13 against a2-allocation budget) and `TestShapeTextLineNativeFallbackAndGraphemeEllipsis` (width24). The Windows package passes7.911s in that full run; the old replacement-directory failure is gone. [Linux vet](evidence/preview-pass252-linux-vet.txt) still reports `internal/linux/browser_seat.go:55:23: possible misuse of unsafe.Pointer`, identical to Round107's baseline. [Separate exit status](evidence/preview-pass252-baseline-status.json) records both failing commands as1. The wrapper intentionally finishes after recording those expected nonzero results; it is not evidence that those commands passed. Generic GUI e2e tests remain disabled in this full run; the explicit native fixture above is the real WebView2 acceptance.

All affected app preview suites pass: [HTML/PDF/OpenUI/WebView/browser-preview checks](evidence/preview-pass252-app-tests.txt). [Command selection](evidence/preview-pass252-command-tests.txt) passes the Host/Preview/Isolation/Transient/Browser/Caption/NativeConversation cases. App preview packages and command vet pass. No bridge or shader changes require JS/native shader regeneration.

Default and pass252 executable SHA256 `BAC54C66F39DBB0239770516D898EF4E56F42631B77D444F792815F7829696CE`; size85,158,912bytes. [Build identity](evidence/preview-pass252-build.json) records both paths. Executables remain untracked build output. The default whaleisle protocol destination remains unchanged.

## Limits and remaining work

This is a correctness repair for the native preview lifecycle, not an Electron visual comparison. The checked filesystem is the host's local Windows temporary volume; ReFS behavior and network filesystems were not executed. Unsupported file-ID queries refuse initialization. The OS identity and `os.Root` safeguards do not claim a hostile-local-user security boundary: directory creation-to-first-open, mutable ancestors/permissions and arbitrary mount-point mutation remain separate concerns.

No deletion runs while a potentially live browser is known to own the folder. Aborted-before-browser or uncertain/in-use folders may remain. A killed process cannot guarantee asynchronous cleanup. The full production Host boot/shutdown path was not repeated; its previously observed shutdown hang remains open.

[Process check](evidence/preview-pass252-processes.json) records protected original ElectronPID20012/backend16008 still running and no remaining native-fixture processes. It also preserves the exact identity of one orphan WebView from the rejected04:12 fixture run, whose parent test process had exited; that verified fixture-only process was stopped during final cleanup. No original process was restarted, signalled or modified. New model requests:0.

API references used for this implementation: [full128-bit file ID](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_id_info), [handle disposition access](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-setfileinformationbyhandle), [one-byte disposition BOOLEAN](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_disposition_info), and [WebView2 browser process lifetime](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-model).

Inspector typed target ranking/parts, picker/graph/lazy/cycle/paging behavior, complete shell/control/font/material parity and performance budgets remain open. The overall parity goal stays active.
