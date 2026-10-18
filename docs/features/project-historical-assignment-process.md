# Exact historical assignment and original process

The native report reader, delivery source-report reader and request Activity
source-report reader now offer **Locate this report's original assignment**.
The source report, its original request/version and the parent page remain
intact behind the nested reader. Back returns to that exact source page.

## Read and address contract

The first action reads only existing `dsh-project/workstreams/get` with
`{projectId, workstreamId}` and `dsh-project/detail {projectId}`. It finds exactly
one receipt matching the selected report's `delegationRef`; it does not select
`currentDelegationRef`, `latestReport`, the current requirement or current brief.
The full returned receipt fields are displayed: ref, messageId, runId,
createdAt, phase, scope, requestId and requestRevision. Report ID, original
request/version, source run/member expectations, Project/work and member
ownership are checked. Missing or ambiguous original entities are explicit
errors. An unavailable original Session never causes a replacement to be made.

Only the separate **Open and locate original process** action follows the exact
original child address:

`{kind:"subagent",parentSessionId:<original Lead>,childSessionId:<original member>,mode:"continuable"}`

The subagent follow observes and validates the original Session; unlike an
ordinary prepared Session, it is not promoted for activation. It sends no
input, invokes no tool/model endpoint, creates no replacement and does not
restart an old run. The opening snapshot header must name the original child,
subagent origin and original parent before any records are accepted. Its `session/follow` carrier is released immediately after
the opening snapshot. Subsequent `session/page` reads keep that snapshot's
`throughSeq`, use the oldest event as `beforeSeq`, and keep the same address.
There is no automatically retained live stream or reconnect in this reader.

## Exact historical range

Like the original UI, the locator searches for the receipt's messageId in a
user/message `data.id` or `data.source.messageId`, loading older pages as needed.
It starts at the preceding turn/start and ends before the next Lead-originated
team-message assignment (including its preceding turn/start). Only the original
UI's turn/start, user/message, assistant/message, tool/call, tool/result and
turn/end event types are displayed, with full JSON bodies. Local pages contain
50 records; no per-record body is silently truncated. If no later assignment
boundary exists, the UI explicitly says the range ends at the frozen snapshot,
not that the historical run completed.

No-progress pages, conflicting event identities, different cursors, absent
assignment messages and missing member Sessions fail rather than substituting
a recent process window. Search is bounded by its 60-second owned context,
20,000 events and 32 MiB of event bodies; a limit error does not show a partial
range as a complete assignment. Failed rereads retain the last complete body.

Host/client, Project, source reader instance and navigation generation, report,
request version, work, delegation, optional recorded run/member, original
message and Session cursor remain bound. New callbacks carry their rendered
view token. Close, main-window destruction, DSH shutdown, collapse, source navigation and Host changes cancel pending
reads and fence late replies. Shared dshClient/dshMux access uses dshMu before
project-state locking. No Host paths, files or external links are opened.

## Fixed original evidence

ChisaAlter/WhaleIsle at e9f584be6d3b491ce777bb6823f30bc2bbc6f195:

- vendor/dsh-project/lib/service.js, blob
  83ae9045a68a8b222ae6d97572a3eb6e4f701f06, lines 345–351: Project-scoped
  workstreams/get and each historical receipt's actual exposed fields.
- vendor/dsh-project/client/client.js, blob
  412aac7679f65541b60faaddeac69ef135bf97c4, lines 152–162: message lookup and
  assignment turn boundaries; 967–1004: backwards loading until the original
  assignment is found, with no-progress/missing guards; 1059–1076: exact
  delegation selection and explicit missing receipt; 1352: original continuable
  child Session address.

- vendor/deepseek-harness/packages/api/session-controller/src/history.ts, blob
  07fbbfe6beb4bf71084c94bab3b57024060f8e75 (same pinned commit), lines 193–212: snapshot header and ordinary-session-only promotion;
  243–269 and 337–379: observation and subagent parent/mode validation.

One new in-memory fixture runs once with `-count=1`. No existing docs/history/
Activity suite is rerun. No real Host, project data, file, model/tool or actual
GUI is accessed. The sole integrator owns incremental Windows main compilation.

Integrator-requested header validation and shutdown hooks were added after the
one fixture run, statically checked only; no fixture was repeated.
