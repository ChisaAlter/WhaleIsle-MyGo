# Native compact pet chat

The desktop pet has two independent entry points: **Quick chat** opens a native
compact card, while **Main pet chat** retains the existing main-window navigation.
Both use Whale's actual persistent pet session, message history and selected model.
Opening the compact card does not replace the current main conversation/composer.

The compact card exposes shared history, the live model/reasoning catalog, Send,
Cancel wait, Stop current turn and an explicit main-session action. Model changes
are session-local plugin changes. No shadow history, synthetic reply or fallback
provider is used. Draft edits survive polling and connection invalidation; the
card must be reopened to use a replacement Host. Disabling chat closes the card.
Autonomous speech and wandering pause while the compact card is open.

Cancel wait cancels only the local HTTP wait; the Host can continue or have a
queued message. Stop current turn calls the public session/cancel operation for
the confirmed pet session and reports its accepted receipt. That endpoint does
not identify a particular request and cannot remove a queued message. The UI
leaves queue management to the main session and does not claim exact-message
cancellation. The plugin's pet/chat and select-model operations also do not accept
an expected-session argument. Native identity checks bracket these calls but
cannot make the server's ensure/mutate sequence atomic.

Fixed authenticated RPC routes, redirect refusal, request serials and
Host/window/session identity gates protect late results. Polling runs only while
the card is live. Tests are provided as source only; no test suite, real Host,
model, GUI or application execution was performed during authoring.
