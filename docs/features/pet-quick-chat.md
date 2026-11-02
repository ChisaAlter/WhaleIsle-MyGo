# Native compact pet chat

The desktop pet has two independent entry points: **Quick chat** opens a native
compact card, while **Main pet chat** retains the existing main-window navigation.
Both use Whale's actual persistent pet session, message history and selected model.
Opening the compact card does not replace the current main conversation/composer.

The real creation path now opens a frameless 248×360-DIP card (minimum
248×220), with a draggable title and an explicit close control. It follows the
character's logical rectangle rather than the full display overlay, with an
8-DIP gap and 4-DIP work-area inset. Its editor and Send control stay visible at
the minimum height; secondary explanatory lines collapse at that height.

The shared pet geometry now selects one display by the reference's fixed
120×130-DIP nominal centre, then its anchor and the primary display. On Windows
the fixed offset uses the anchor monitor's scale, independent of the resized
stage; this replaces the temporary anchor-first workaround without reintroducing
100%/150% selection oscillation. Placement reads the actual HWND origin in
physical pixels, and scales the card gap/inset once. The pet overlay's synchronous
programmatic cross-screen move temporarily permits movement, then restores its
move restriction; this avoids the toolkit blocking SetWindowPos while retaining
the intended user interaction rule.

On 2026-10-26 an owned isolated pet window was placed on real 2560×1440 at 150%
and 1920×1080 at 100% displays. Both settled to the same 240×260 CSS stage, with
matching HWND scale/browser DPR 1.5 and 1 respectively; repeated centre-boundary
placement and return passed. The separate compact chat creation still measured
248×360 DIPs. Its own mixed-DPI card drag/focus interaction has not been exercised
on every display. See the renderer parity note for the physical/CSS velocity
conversion, migration marker round-trips, actual measurements and remaining OEM,
scale-combination and continuous-gesture limits.

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
the card is live. Focused package/protocol tests, minimum-height content layout
checks and an isolated real Windows GUI creation check have run. The latter
measured the actual 248×360-DIP HWND and captured only its owned window, using a
fresh profile with no user process or physical input control. A real authenticated
Host conversation and live send/model-switch/cancellation acceptance remain
outside that GUI fixture.
