# Native page keyboard preferences

The main conversation shell routes these defaults to its existing button actions
(Command replaces Ctrl on macOS, except terminal is unbound there):

- Ctrl+B: collapse/expand the left sidebar, restoring its last dragged width
- Ctrl+N: new conversation, using the existing blank-session reuse behavior
- Ctrl+O: browse/open a workspace
- Ctrl+,: settings
- Ctrl+\\: toggle the right tools/surfaces pane
- Ctrl+Alt+B: toggle the right pane (alternate upstream command)
- Ctrl+`: toggle the terminal drawer (Windows/Linux only; macOS leaves it unbound because Command+` switches windows)

Source of truth: Electron's `ui-workspace/src/client/shortcuts.ts`,
`ui-settings-general/src/client/index.ts`, `ui-titlebar/src/client/apply.ts`,
and `ui-sidebar-right/src/client/shortcuts.ts` in the 2026-10-13 reference.
The older feature-document summary describing Ctrl+B as the right sidebar is
superseded by those actual bindings: Ctrl+B is the upstream LEFT sidebar.

## Input ownership

These are native main-page commands, not system hotkeys or guest WebView
accelerators. Editable/selectable text, IME, custom input widgets (including
terminal input), dialogs and popovers retain their keys. No bare character keys
or common text-editing chords are registered. Held command keys do not repeatedly
execute a navigation action. Dispatch-time guards protect focus changes between
frames, rather than relying only on the last rendered focus state.

## Native preferences

Settings → Interface → Main-page shortcuts lists seven bindings for six actions;
the two tools-pane rows intentionally invoke the same action. Search matches the
Chinese/English action label, native ID, draft or default binding. Each row shows
its saved/default value and a draft field, Unbind and Use default. Draft edits,
unbinding and single-row defaults apply only after Save shortcuts succeeds.
Reset all requires a separate Restore and save confirmation; Cancel writes nothing.

Enter Mod plus a letter, digit or supported punctuation, optionally Shift/Alt.
Mod is Ctrl on Windows/Linux and Command on macOS; Ctrl/Cmd are portable aliases.
Invalid, duplicate and reserved chords show issues and block saving. Editing
chords, existing file-save keys and system/window controls are not offered. On
macOS, Command+` and Command+Shift+` remain window-switching keys, the terminal
default is unbound, and screenshot/Dock system combinations are rejected. No
system hook or guest-WebView interception is registered.

Only the current native binding IDs are stored in config.json → goPageShortcuts.
An absent ID uses its platform default; an empty string explicitly unbinds that
row. Unknown config keys and unknown entries within goPageShortcuts are preserved.
Successful atomic replacement publishes a new shared active snapshot used by the
main shell on its next render and loaded on restart. A failed write leaves the
previous active bindings and the draft intact. A changed on-disk config is refused
with the existing config conflict error; restart after resolving the external edit.
Invalid or conflicting persisted rows are disabled, and unreadable preferences
fail closed instead of silently overwriting them.

This is a bounded native implementation. It does not read, modify or migrate
Electron's keybindings.json, register its broader command set, capture key chords,
or add actions that the native shell cannot execute. Search-focus and other
unimplemented upstream actions are not mapped to unrelated callbacks.

## Directed fixture

`go test ./internal/shortcuts ./internal/config ./internal/ui/frame ./internal/ui/settings -run '^(TestDSK10|TestPageCommandsRespectInputOwnership)' -count=1`
checks native form actions, validation/conflicts, failed-save drafts, explicit
reset confirmation, config preservation/reopen/conflicts, rebound callback
dispatch, exact modifiers, text/IME/custom-input ownership and modal recovery
without starting a Host. macOS reserved-key validation is a deterministic policy
fixture; it is not evidence of macOS hardware acceptance.

## Left sidebar preference

The expanded sidebar remains 264–420 DIP. Collapsing keeps a 56-DIP rail with an
expand button; the expanded header has a collapse button. The compact rail has
no draggable divider. The last expanded width and collapsed state are saved in
the existing go-resume snapshot; older snapshots fall back to the configured
expanded width. Sidebar toggling retains the center pane's tree identity.

`go test ./internal/ui/frame -run '^TestSidebarCollapseRestore$' -count=1`
checks actual shell toggle buttons, Ctrl+B, divider resize, geometry, resume
serialization and text/IME/custom-input ownership without starting a Host.
