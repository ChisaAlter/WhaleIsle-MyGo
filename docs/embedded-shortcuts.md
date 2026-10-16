# Product iframe shortcut routing

## Exact original behavior

At e9f584be6d3b491ce777bb6823f30bc2bbc6f195, Electron
src/main/shortcuts.js before-input-event consumes an accepted matching physical
key and sends kind iframe/webview plus embedding frameName to the product
renderer. The original shortcuts/src/client/native.ts is not merely a pause
filter: it verifies activeElement, isConnected, frame selector and name before
calling the registered product command's dispatcher.

A real current consumer is ui-sidebar-documentpreview/src/client/html/HtmlBody.tsx:
its interactive and basic HTML views use named data-html-preview sandbox
iframes. The browser sidebar iframe selector also has original browser E2E
coverage. A standalone MyGo LocalPreview window is a different path and remains
outside this capability.

## Implemented path

The product-only exact-origin preload offers a read-only guestKeyboard
subscription. Only the build-adapted shortcuts service subscribes. An independent
tracked frames patch adds that subscription to the original native dispatcher;
its native.ts modification narrows the parameter type to subscribe-only. It
adds no close-window capability and does not create dshDesktop.

The accepted product key adapter enables a frame only when the original
subscriber is ready, the top product has focus, no modal/recording/composition
owns input, and its focused connected iframe has a nonempty name and one of the
two original markers. Accepted keys still originate from the product registry.
A native physical-key result must match the accepted revision, key tuple,
frame object and captured name. The adapter then delivers the gesture to the
original native.ts consumer, which repeats its live ownership/revision checks.
It carries no command ID. Main-frame DOM/held-key dispatch stays unchanged.

No frame contentWindow/document is read, no postMessage is trusted, and no
script or RPC is injected into a guest or independent local preview. Navigation,
name/element changes, disconnection and subscriber teardown revoke delivery.
The one-turn focus recheck handles the browser's iframe-focus event ordering.

## Build and verification

current-host-shortcuts.mjs composes provider, held-key and frame adaptations in
that order. Each has its own exact-source provenance record. The provider
record retains the pre-frame provider checksum and verifies the combined final
source checksum, so repeated preparation is idempotent.

The Node fixture runs the actual original native.ts dispatcher with only TS
syntax removed. It verifies accepted iframe routing, changed names, detached
frames, command/menu rejection and subscription revocation, without accessing
guest documents. The combined adaptation fixture and existing Go product
keyboard ownership contract pass.

Live Windows GUI verification is still pending. The physical path has the
preceding module's Windows WebView2 accelerator coverage; ordinary unmodified
DOM-only keys inside an opaque iframe cannot use the main-frame DOM held-key
fallback. Electron webview guests are not claimed as implemented by MyGo.
Independent local previews and arbitrary iframe content remain isolated.
