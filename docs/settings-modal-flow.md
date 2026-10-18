# PR05: settings overlay and workspace return

Settings now renders as a centered, viewport-bounded modal above the existing workspace instead of replacing its content. The navigation/workspace tree stays mounted, retaining composer and file-editor state. The shared navigation settings action no longer clears the active market/schedule/assistant route before opening settings.

The existing `ui.DialogBase` supplies a dimmed backdrop, modal keyboard ownership, Tab containment and focus return to the invoking control. Backdrop click and Escape call the same existing settings `OnClose` callback as the settings close button. Existing settings save/reload behavior and the callback's draft/reset policy were retained rather than adding an alternate discard path.

Native browser/plugin/account child surfaces are hidden while the canvas settings modal is visible, without closing their owned pages or revoking account leases. The ordinary view rendering restores browser surfaces; the same still-owned plugin/account Host is shown again when settings closes. No page navigation is performed by this visibility helper.

Verification: synthetic modal Escape/focus-return and workspace-draft preservation fixture, viewport sizing fixture, and Windows amd64 command package compilation. Actual Windows/macOS visual geometry and native-child focus behavior remain desktop runtime acceptance items. No motion or typography refinement was attempted.
