# Native recorded Project reports
Fixed official source: upstream-90465b67/vendor/dsh-project/lib/service.js listStore/resultHistory (934-956), client/client.js ResultCard/ResultHistory/ResultEntry (754-795).

Previous native process history read original member session events, not recorded delegation reports. This addition covers the missing report material browser:
- Project button -> store/list results, cursor pagination.
- Work entry -> store/results workstreamId and cursor, previous/older pagination.
- Report selection -> store/results exact delegationRef plus existing WorkDetail identity validation.
- Read-only full report summary, evidence, issues, exact delegation/call/time and request/version metadata. Current delegation is compared against actual work detail, never inferred from idle.
- Original work details/process reuses existing Team/detail UI after reading TeamOverview; it is original session navigation, not a filtered historical snapshot.
- File opening reuses ResolveArtifact with exact work/delegation identity and frame OnOpenDeliverable dirty-editor guard. Changed-file notice and current-content warning; no historical diff or rollback.
- Errors retain current page; cursor changes apply only after successful read. Host/client generation and selected-session/navigation guards prevent stale file opens.
- One focused mock fixture TestRecordedReportHistory passed. No real Host/session/model/data touched. Windows compile delegated to integration owner.
