// Package openui renders generative UI (OpenUI Lang, "Whale UI") natively:
// the element tree of an internal/openui/lang Frame becomes coss
// components (internal/ui/coss) styled by the design tokens
// (internal/ui/design). No web engine runs on this path; only HtmlView,
// YouTube and other components without a native view show a placeholder
// whose button opens the card in the A2 compatibility workbench
// (ARCHITECTURE.md §7, 2026-11-23).
//
// # Model
//
// A card is a Runtime, implemented by internal/openui/card.Card: it owns
// the lang.Instance, parses and evaluates only when the program or its
// state changes, persists the interaction state (openui-state-v1) and runs
// actions' host effects off the UI thread. View reads the card's last
// Frame and never parses, evaluates, does I/O, starts goroutines or
// timers.
//
//	card.Card ──Frame()──▶ View(c, Options) ──Node(el)──▶ registry[el.Type](r, el)
//	   ▲                                                       │
//	   └── SetBinding / SetField / Dispatch(revision, …) ◀─────┘ (input, real clicks)
//
// # Components
//
// A Component builds one element:
//
//	type Component func(r *Render, el *lang.Element) ui.Element
//
// Register(name, fn) installs it; Native lists what is installed. Node
// keys each element by el.Identity() so per-element UI state (ui.Local)
// follows the element across frames. Content builds a prop that holds a
// string, an element or an array of both; Children builds a children prop
// in a vertical container and Within marks horizontal ones, so Inline
// keeps compact controls at their own width in columns.
//
// Inputs keep a local value and write through the card: a $binding with
// SetBinding, a plain value as a form field named by its label (Whale
// viewer semantics). A state value written elsewhere replaces the local
// text; the user's own typing is never reformatted.
//
// Actions run only from a real click: Dispatch carries the frame revision
// the click was built on, and the card ignores clicks on an older revision,
// clicks while the element's action runs and clicks while the program is
// still streaming. Failures show as errcopy sentences under the control;
// diagnostics of a frame that cannot render show as one classified
// sentence. Raw errors never reach the screen (docs/error-copy.md).
//
// Pictures come from the media service (internal/desktop/media) through
// Runtime.Image and SearchImage; the bytes come from the public image
// adapters of internal/desktop/openui, called as Go functions.
package openui
