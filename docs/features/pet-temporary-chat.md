# Explicit temporary API pet chat

When the compact card is opened while the desktop already reports Host unavailable,
or the Whale assistant is disabled, it offers **联网发送 / Send to API**. This is
an online request using the desktop's existing `apiKey` / `baseUrl` and the
original `deepseek-chat` model. It is not a disconnected/local model. Merely
opening, painting, refreshing status, notifications, care or sleep never sends.

The native process resolves the existing `credentials.Load` only after explicit
Send, including its existing safeStorage support. It uses the same
`<DSH_HOME>/data/whale/settings.json` name, title and additional persona; an
explicitly saved pet personality wins. No catalog-provider key is substituted.
Credentials never enter the pet HTML, action URLs, UI status or logs.

This mode is distinct from the persistent Whale session. Six successful
user/assistant pairs are kept only in the card's memory and erased on close.
Neither history nor drafts transfer between modes. Interrupted direct cards
start fresh. A late reply cannot overwrite another draft, cancelled request,
closed/recreated card or changed Host generation/origin/cookie. Provider, key or
persona changes require closing/reopening before the old memory could be sent.

Shared-session errors, timeouts and uncertain sends never trigger a direct
request. Unlike upstream's automatic transport fallback, this deliberately
requires unavailable state before opening and an explicit, clearly labelled API
Send. If the Host appears ready but an RPC fails, the shared card reports that
failure; it does not guess that a second delivery is safe. Host recovery or
replacement invalidates the temporary card; reopen to return to the shared chat.

Requests use the original 30-second timeout, 2,000 UTF-16 input limit,
`max_tokens: 360`, 240-unit reply cap and six-pair memory. Non-loopback cleartext
HTTP, embedded URL credentials, query/fragment endpoints and redirects are
refused. No streaming, retry or Host cookie forwarding is introduced. Cancel
only cancels local waiting and cannot guarantee that the API did not receive or
charge for the request.

Source: ChisaAlter/WhaleIsle@e9f584be6d3b491ce777bb6823f30bc2bbc6f195,
`src/main/pet-chat.js` and `src/main/desktop-live2d.js:354–385`.
Focused fake-transport and UI-wiring fixtures cover the local contracts; actual
Windows UI, saved credentials and a real model request were not exercised.
