# Native page keyboard commands

The main conversation shell routes these defaults to its existing button actions
(Command replaces Ctrl on macOS):

- Ctrl+B: collapse/expand the left sidebar, restoring its last dragged width
- Ctrl+N: new conversation, using the existing blank-session reuse behavior
- Ctrl+O: browse/open a workspace
- Ctrl+,: settings
- Ctrl+\\: toggle the right tools/surfaces pane
- Ctrl+Alt+B: toggle the right pane (alternate upstream command)
- Ctrl+`: toggle the terminal drawer

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

## Deliberately not implemented here

No custom-binding editor, keybindings persistence, guest interception or separate
command registry. Search focus and other commands without
an existing matching shell action are not falsely mapped to another action.

## Directed fixture

`go test ./internal/ui/frame -run '^TestPageCommandsRespectInputOwnership$' -count=1`
checks callback dispatch, exact modifiers, plain-key pass-through, text/IME,
custom-input ownership, and modal suppression/recovery without starting a Host.

## Left sidebar preference

The expanded sidebar remains 264–420 DIP. Collapsing keeps a 56-DIP rail with an
expand button; the expanded header has a collapse button. The compact rail has
no draggable divider. The last expanded width and collapsed state are saved in
the existing go-resume snapshot; older snapshots fall back to the configured
expanded width. Sidebar toggling retains the center pane's tree identity.

`go test ./internal/ui/frame -run '^TestSidebarCollapseRestore$' -count=1`
checks actual shell toggle buttons, Ctrl+B, divider resize, geometry, resume
serialization and text/IME/custom-input ownership without starting a Host.
