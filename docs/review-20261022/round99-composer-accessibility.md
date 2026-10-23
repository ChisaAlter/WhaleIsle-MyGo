# Round 99 — Accessible composer value and edit commands

## Confirmed problems

The custom composer was editable by native text services, but its accessibility node exported an empty value and no SetValue action. The engine only populated those from its internal TextArea editor; the composer uses its own TextInputClient and undo owner. In pass242, the actual Windows UIA Composer draft Edit was enabled but ValuePattern reported IsReadOnly=true and Value="". This did not prove that physical keyboard input was read-only.

While adding a full edit-history regression, another omission was reproduced: Ctrl/Cmd+Z/Y had handlers, but the system Edit menu's `undo` and `redo` commands did not. They now use the same DraftClient history and preserve the existing composition guard.

## Implementation

- Add optional `ui.AccessibleTextInputClient` metadata: placeholder, multiline, read-only and password. Only clients explicitly implementing this contract opt into full-document accessibility reads and writes. Legacy clients retain their explicit element Value and previous behavior.
- Read the current custom client directly for an accessibility snapshot, so unfocused controls remain readable. Convert UTF-16 client selection endpoints to the AccessNode protocol's rune offsets. Password snapshots contain neither text nor selection and never query the document.
- SetValue goes through one registered-client ReplaceText call, using the documented whole-document range that the client clamps. It preserves client-owned undo and mutation behavior. No shadow TextArea, synthetic select-all, direct draft assignment or copied history is introduced.
- Reject writes to removed, hidden, inert, disabled, modal-obscured or retained inactive router controls. Recheck client/adapter identity after native reset can reenter the host.
- Reset native text services before a full-value replacement, including for the same focused client, so a pending preedit range cannot be applied to the replacement document. Preserve the existing built-in editor path.
- The composer's stable typingInputClient inherits metadata from DraftClient. Current localized placeholder and multiline state reach accessibility without changing the painted layout.
- Add `Tester.SetValue` to drive the actual accessibility action in application tests, and document the optional contract.

The custom accessibility snapshot includes marked text, consistent with TextInputClient's document contract. Windows still exposes ValuePattern at the built-in TextArea level; this change does not add TextPattern or native character-range navigation.

## Verification

Passed:

```text
go test ./ui -run '^TestAccessibleTextInputClient|^TestAccessibility|^TestTextInputClient|^TestReadOnlyInput|^TestTester' -count=1
go test ./internal/ui/composer ./internal/ui/frame -count=1
go test ./cmd/whaleisle -run '^TestWorkspaceCaption' -count=1
go vet ./ui
go vet ./internal/ui/composer ./internal/ui/frame
CGO_ENABLED=0 GOOS=windows go build ./ui
CGO_ENABLED=0 GOOS=linux go build ./ui
CGO_ENABLED=0 GOOS=darwin go build ./ui
CGO_ENABLED=0 go build -o whaleisle-pass243.exe ./cmd/whaleisle
git diff --check
```

After adding explicit keyboard undo/redo assertions to the application regression, this also passed:

```text
go test ./internal/ui/composer -run '^TestNativeDraftAccessible|^TestNativeDraftInputRetains|^TestNativeIME' -count=1
```

Coverage includes unfocused/background-window snapshots and writes, UTF-16 emoji selection conversion, legacy/read-only/password/disabled behavior, hidden ancestors, modal scope, removed/closed controls, retained router pages, client replacement during native reset, end-preedit followed by SetValue and a delayed native commit, and hero/conversation composer layouts. Application regressions verify one full replacement, menu and keyboard undo/redo, Send consuming the exact multiline value, cleared sent history, marked-text cleanup, subsequent native typing and inert composer rejection.

### Existing toolkit failures

A full `go test ./ui -count=1` has two failures. Exporting HEAD before this round into an independent ignored baseline directory and running those tests reproduces the exact same results:

- `TestDiffFrameAllocs`: steady frame 6 allocations, warm scrolling 13; expected at most 2.
- `TestShapeTextLineNativeFallbackAndGraphemeEllipsis`: width 24 does not use the expected native one-line ellipsis.

They are not introduced by this round. The full toolkit suite is therefore not reported as passing.

## Windows live evidence

Build pass243 runs with the existing isolated `.devdata-clean` profile (PID 17976 at verification). The default whaleisle executable was updated after a successful build.

- pass242: enabled Composer draft Edit, ValuePattern IsReadOnly=true, empty Value.
- pass243: enabled Composer draft Edit, ValuePattern IsReadOnly=false.
- SetValue with `输入回归测试 😀\n第二行：保留完整草稿` succeeded. A later ValuePattern read returned the exact complete string and the rendered text contains the expected two lines: [draft screenshot](evidence/composer-accessible-pass243.png).
- Replaced the draft with `请只回复：MYGO_NATIVE_COMPOSER_OK。不要调用工具、访问文件或运行命令。`, verified an exact readback, then invoked the exact enabled Button named 发送消息 once.
- The actual conversation contains exactly one copy of the user prompt, the composer is empty and remains accessible/editable: [send screenshot](evidence/composer-sent-pass243.png).
- The selected deepseek-official route has no API key in this isolated profile. The Host returned a missing-credential error, which is shown in the UI. There is no model response and no successful model roundtrip is claimed. No credentials were read, copied or changed.

### Live keyboard automation limitation

The first SendKeys attempt did not change the value. Subsequent inspection showed `SetForegroundWindow` returned false and the foreground HWND differed from the app. A second attempt explicitly gated all keystrokes on successful foreground activation; Windows again refused, so it stopped before sending any keys. This is not evidence of an application Ctrl+Z failure or a passing physical keyboard test. Headless key-event and native text-client regressions passed; foreground physical typing remains unverified in this environment.

The first cold-start UIA query also ran before the Composer draft was present and failed as missing; later enumeration and the successful read/write/send verification used the mounted control. No startup failure is claimed.

## Remaining parity work

- A configured route is still needed to verify a successful real model reply and its streaming behavior.
- The missing-key state currently exposes the raw provider error in a top strip as well as the localized explanation. The live send screenshot also exposes internal context rows. These visible surfaces need same-state comparison with Electron.
- Exact same-state visual/motion coverage across the application and remaining Git confirmation semantics remain unfinished.

The overarching parity goal stays active.
