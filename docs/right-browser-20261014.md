# Browser in the workspace right seat
Evidence: fixed Electron src/shared/single-right-panel-contract.test.js and vendor/deepseek-harness/packages/client/ui-preview/src/client/apply.ts register BROWSER_KIND in the sole rightbar. MyGo's right-pane Browser button previously called showBrowserPreview and left the conversation for a full-page panel.

Delivered:
- Right pane Browser is an active tab next to Files/Diff/Agents. It renders the existing Controller in that seat, keeping chat/composer and the existing terminal column mounted.
- Existing Controller owns pages/history. Switching Files/Diff or closing the seat hides child windows; does not close tabs, reset file buffers, or stop terminal processes. Return control selects Files.
- Explicit global full-page browser remains available and resets the viewport mode; no duplicate browser implementation.
- WebView Host supports optional parent-content-relative viewport. Move/resize stays bounded in the seat, not full-width. Existing unrestricted/full-panel Host behavior remains unchanged with nil viewport.
- Empty/unlaid-out browser seat stays hidden until bounds exist.
- One synthetic fixture TestDockedViewportKeepsRightSeat passed for seat geometry, parent move/shrink, full-panel restoration and empty geometry. No real page, Host/model, file, terminal or Windows UI was run.
- Three new files + integration.patch. Apply check passed. Windows cumulative compilation delegated to integrator; live Windows child-window rendering still needs actual-device verification.
