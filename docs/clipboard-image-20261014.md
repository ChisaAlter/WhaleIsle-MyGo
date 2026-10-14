# Explicit composer clipboard-image attachment
Source contract: Electron vendor/deepseek-harness/packages/client/ui-conversation/src/client/input/editor/keymap.ts PASTE_COMMAND (file intake plus independent text paste). Native draft_input.go previously read only plain text.

- Input paste calls image intake and preserves nonempty plain text paste. Image-only/empty clipboard does not erase the selected draft text.
- Explicit Paste image button provides the same image operation without altering text. Empty clipboard reports no image only for the explicit button.
- OS Clipboard.ReadImage is invoked only by that user action, never polling/background capture.
- PNG bytes are bounded to 16 MiB / 24 megapixels, fully decoded then written as a new private cache attachment. Invalid/empty input leaves no cache artifact.
- The original draft pointer, session ID, edit sequence, session store and Host identity are captured. Late save after navigation/connection/edit change removes only its newly created cache file and attaches nothing.
- Uses existing addComposerFiles -> PrepareComposerAttachment -> submitComposer, retaining the existing attachment/remove/vision/send/error/recovery behavior. No new upload/send RPC, no implicit send.
- Focused synthetic TestClipboardImageIntakeFixture passed. No actual clipboard read, screenshot, Host or model call performed. Windows cumulative compile delegated to integrator.
- Four new files + minimal main/frame/composer/paste integration.patch. Apply check passed.
