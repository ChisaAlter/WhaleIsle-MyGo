# Native usage project ranking
Source: fixed Electron vendor/dsh-usage-panel/src/shared/contract.ts PageRequest/ProjectRow/ProjectPage; host/index.ts projectsMore; client/components/MoreRankModal.tsx projects mode.

Code inspection confirmed native ranking.go only had session and model ranking. Added:
- Actual projects.more request {offset,sort:"tokens"|"cost"}, receiving rows/hasMore. No invented prices, aggregation, Project IDs or date-range API.
- Project means session working directory; null directory remains a distinct legitimate bucket. Full path displayed.
- Explicit initial load/refresh, server-side tokens/cost sort reset to zero, more/retry. Failed pages keep rows and exact cursor; dedup display doesn't change wire offset.
- Single pending page; reconnect/detach/overview refresh and saved billing-price refresh invalidate pending pages. Date selection hides all-time project ranking instead of relabeling it selected-range.
- Estimates reuse saved Billing settings and official per-period model buckets; incomplete or unloaded pricing displays unpriced. No guessed zeros or vendor rates.
- Single targeted mock fixture TestProjectRankingFixture passed (route/auth, initial/null bucket, retry offset, dedup, sort reset, duplicate suppression, detach and unknown costs).
- No real Host/data/model. Windows cumulative compile delegated to integration owner.

Three new Go files + integration.patch for Store field, reset hook, billing reset and statsView. No changes to existing session/model ranking behavior beyond independent reset hook.
