# Product shortcut priority: macOS and Linux

## Implemented behavior

The existing authenticated product-page accepted registry now receives real native key-down and key-up candidates on macOS/Linux through the same `ProductWebKeyEvent` / `ProductWebKeyReset` contract as the Windows held-key implementation. A true return consumes that event before normal WebView handling. Unmatched keys pass through. No command names, application RPC bridge or website opt-in is added.

- macOS: the existing public `NSWindow` subclass overrides `sendEvent:` and `performKeyEquivalent:`. Candidates are considered only while the first responder is the WebView or its descendant. `NSEvent.keyCode`, modifier flags and `isARepeat` are forwarded; the responder's public `NSTextInputClient.hasMarkedText` protects composition. Blur and destruction reset held state.
- Linux: GTK top-level `key-press-event` / `key-release-event` signal handlers run before the default GTK handler and menu handler. Focus must be inside the WebView. The existing registry decides whether to consume. A native down set supplies repeat information; a queued identical-time release/press pair is recognized as X11 auto-repeat. Window blur, composition and leaving WebView focus reset held state.
- Linux observes the public WebKitGTK 2.28+ input-method context, preedit signals and current preedit text. It never runs the input method's filter a second time. Missing input-method observation APIs disable native priority instead of guessing. The application retains its existing renderer handling.
- Physical positions map to the registry's existing VK-shaped identifier. macOS uses known NSEvent key positions; Linux uses the known subset matching WebKitGTK's own hardware-keycode-to-DOM-code mapping. Locale-dependent `keyval` is never mistaken for `KeyboardEvent.code`.

The application-owned accepted snapshot remains responsible for editor/input/composition/plugin ownership. Its navigation, crash, close, inactive-owner and revision revocation are unchanged. The accompanying shared `SetProductKeyboard` change invokes optional `ResetProductKeyboard` on replacement: Linux revokes consumption while retaining the presence of already-held keys until release, so a repeat cannot seed a new owner's chord. This adapter depends on the held-key module's paired-event API and snapshot hook.

## Boundaries

Ordinary Browser-profile windows and isolated document/recording helpers never enter this path. Other windows without an active authenticated accepted snapshot do not execute product commands. Native focus and composition checks supplement, rather than replace, the product's ownership protocol.

macOS Option-only combinations pass through because they can begin dead-key composition before marked text is visible. Linux nonzero keyboard groups, AltGr/Level3/Level5, dead and compose keys pass through. Unknown/keypad/international positions outside the mapping pass through. Known macOS F1–F20 and Linux F1–F24 positions are mapped; actual registry acceptance remains governed by the shared grammar. Custom platform keycode mappings outside stock WebKit conventions are not claimed supported. OS-reserved shortcuts that never reach the application cannot be intercepted.

This change supplies native product priority, not a global hotkey registrar or an expansion of the separate application fallback-shortcut API. It does not modify native IME configuration or give ordinary webpages Host RPC.

## Validation

Feature-focused validation only, CGO disabled:
- Physical-code mapping fixtures and Linux native-routing fixtures passed. They cover physical down/repeat/up, fresh press, snapshot repeat retention, alternate keyboard group, active composition, other-widget focus, missing IME APIs and ordinary Browser exclusion.
- Darwin arm64 and Linux arm64 backend test binaries compiled, not executed.
- Darwin amd64 and Linux amd64 SDK root-package builds passed.

The first Linux fixture failed because it retained a uintptr into movable Go-stack event storage. The fixture was corrected to use static storage representing a native event pointer, then the targeted checks passed. The final snapshot-retention case was checked after the shared hook became available. No unrelated test suite was run.

No AppKit/GTK live keyboard, IME, compositor, real auto-repeat, plugin focus or native default-action consumption test was available. Compiled callbacks and synthetic fixtures are not native desktop acceptance. Real-target testing remains necessary, especially focus/ownership transitions and layout-specific input.

## Public API references

- https://developer.apple.com/documentation/AppKit/NSWindow/sendEvent(_:)
- https://developer.apple.com/documentation/appkit/nsresponder/performkeyequivalent(with:)
- https://developer.apple.com/documentation/appkit/nstextinputclient
- https://docs.gtk.org/gtk3/signal.Widget.key-press-event.html
- https://docs.gtk.org/gdk3/struct.EventKey.html
- https://webkitgtk.org/reference/webkit2gtk/2.40.5/method.WebView.get_input_method_context.html
- https://www.webkitgtk.org/reference/webkit2gtk/stable/method.InputMethodContext.get_preedit.html
- https://raw.githubusercontent.com/WebKit/WebKit/main/Source/WebKit/Shared/gtk/WebKeyboardEventGtk.cpp
