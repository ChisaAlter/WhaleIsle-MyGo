# Native token statistics

Source: Electron `vendor/dsh-usage-panel/src/shared/contract.ts` (overview version 4), `src/client/api.ts`, and current `usage-stats` settings section. The plugin is already built in via MyGo `internal/harness/overlay.go`; marketplace exclusion prevents duplicate installation, not this API.

Delivered: `/usage-stats/overview` with endpoint cookie, forced refresh, 60-second transport timeout, all-time and recent-30-day token buckets/session counts, coverage diagnostics and failed IDs, 7/14/30-day UTC charts, model ranking, top-session navigation, JSON and token-only CSV exports through the native save dialog. JSON preserves the full original payload including billing buckets. CSV uses UTF-8 BOM, RFC4180 quoting and formula guards. Failed refreshes retain explicitly stale prior data; reconnect/detach cancels and discards old responses. Existing account and job controls remain separate; job counts are no longer displayed as usage charts.

This delivery does not claim full plugin visual/feature parity: custom billing-price editing and estimates, heatmap range selection, additional ranking pagination, and destructive log repair remain outside this bounded read-only statistics module. No guessed prices, account changes, log rewrites, plugin reinstall, or real-host requests were performed.

Verification: isolated HTTP fixture covers exact endpoint/auth/force, repeat-refresh totals (no accumulation), contract rejection, loss of backend with cached result, detach isolation, JSON preservation and CSV safety. Run `go test ./internal/store/usage ./internal/ui/usage`. Windows cumulative compile is owned by integration.
