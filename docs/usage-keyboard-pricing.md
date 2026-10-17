# Usage keyboard and price review

The native usage calendar now supports focus-based day details, arrow navigation,
Home/End, Enter/Space selection and Shift+Enter/Space range extension. Existing
click selection, explicit Extend buttons, date range inputs and exports remain.
Because this calendar lays weekdays across and weeks down, left/right moves one
day and up/down seven days. Missing returned dates remain inert. The original
uses the transposed graph, with left/right seven days (read-only reference:
`vendor/dsh-usage-panel/src/client/components/Heatmap.tsx:96-150`).
Focused or hovered dates display the existing per-model token/cost detail below
the calendar; keyboard users no longer depend on a pointer tooltip.

The Price details and settings button opens a focus-trapped, scrollable dialog.
Close, Escape and backdrop dismissal restore the opener; closing preserves the
store's unsaved price draft. Save prices and Discard draft remain explicit and
continue using the existing validation/persistence path. Dismissal never saves.
Reference: `vendor/dsh-usage-panel/src/client/StatsSection.tsx:235` and
`BillingSettingsModal.tsx:424-443`. Existing billing content was moved intact.

Verification: focused synthetic navigation cases passed (day/week directions,
bounds, missing dates, Home/End); actual ui.Tester dialog Escape/focus restoration,
draft preservation and reopen passed. Windows amd64 usage package cross-compiled.
No Windows/macOS desktop execution or visual acceptance was performed. The native
calendar retains ordinary button Tab traversal, rather than the original roving
tabindex. Orientation, animation and pixel styling remain outside this change.
