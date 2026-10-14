# Native usage date filters

Source: Electron `vendor/dsh-usage-panel/src/shared/contract.ts` v4 and `client/components/Heatmap.tsx`.

- Host overview provides 182 UTC daily buckets, inclusive YYYY-MM-DD keys. Native date inputs and day/extend buttons select those returned days; 7/14/30-day presets anchor to the latest host day. Missing days are rejected, never interpreted as zero.
- Selection scopes daily charts, token summary, model ranking, and JSON/day/model CSV exports identically. Clear restores all-time model/session rankings and original full JSON export.
- JSON selection retains exact daily cost/modelCosts and full scan coverage diagnostics. All-time fields are omitted from selected export; scan diagnostics are explicitly labeled full-scan, so failed/pending sessions are not disguised as complete coverage.
- The original heatmap only scopes its card summary. This native UI also scopes token ranking/export as requested. The host does not offer dated session ranking: sessions.more accepts offset/sort only; topSessions is all-time Top-N. Selected UI hides session ranking/export with an explanation; it never synthesizes range sessions from lastActive.
- Billing controls remain a separate explicitly all-time estimate; selected model totals do not invent price-phase attribution.
- Refresh that evicts any selected date clears selection and explains why. Invalid/reversed/out-of-window drafts leave the last valid range intact.

Verification: focused fixture `TestStatsDateRangeSelectionExportAndClear` passed (selection inclusive, model ranking, three filtered exports, unavailable dated sessions, invalid ranges, clear restore). Cumulative Windows build delegated to the integrator; no real Host, full suite, race or UI screenshot run.
