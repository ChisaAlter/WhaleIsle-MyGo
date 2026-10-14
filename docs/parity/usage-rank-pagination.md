# Usage ranking pagination

Baseline: `83e4542328ad7c19a9e00d89dd5a0083a034b76b`.

Source of truth: Electron `vendor/dsh-usage-panel/src/shared/contract.ts`, `src/host/index.ts` (`sessionsMore`, PAGE_SIZE=10), and `src/client/components/MoreRankModal.tsx`.

- Session ranking reads `sessions.more` with `{offset, sort:"tokens"}`. The endpoint covers the all-time session index, not the selected daily range. No fabricated date parameters or `models.more` endpoint.
- Clicking more starts at offset zero and replaces overview Top-N with the paging response. Subsequent clicks append pages; the cursor counts wire rows even if duplicate IDs are suppressed in display. Overview Top-N is never assumed to be the server's first-page length.
- In-flight clicks are ignored. Failed pages keep existing rows and offset and offer retry. Empty terminal pages stop normally; incomplete responses, missing IDs and empty nonterminal pages surface errors without advancing.
- Refresh immediately cancels and invalidates the page generation/cursor, including failed refreshes. Attach and detach do the same. Stale completions cannot repopulate a refreshed or reconnected list.
- Model rows are complete in `overview.allTime.byModel` or computed from selected UTC days. They display ten at a time locally with shown/total counts, with no additional RPC. Refresh and applied/cleared date selection reset their page.
- The native baseline has no project-ranking entry. `projects.more` exists in the original host but is not newly introduced here.
- CSV session export retains the existing overview Top-N contract, explicitly labeled independently of paged rows; this module does not silently claim to export every page.

Verification: one isolated HTTP fixture `TestRankingPaginationFixture` passed, covering exact route/cookie/payload, first page, duplicate click, retry with unchanged cursor, duplicate row suppression, terminal page, refresh cursor reset, delayed response invalidation, empty result, model paging/date reset and detach reset. No real host or log operations. No full/race suite run. Windows cumulative compile delegated to integration.
