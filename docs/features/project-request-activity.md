# Native request activity snapshot and source navigation

Projects → Requests → a request → Activity timeline opens the original
`dsh-project/requests/get {projectId, requestId}` snapshot. The native view
projects the same kinds of timestamped entries as the original Activity tab:
requirement revisions, associated assignments, member reports, native run
settlements and the request's current delivery. Entries are ordered newest
first and missing/nonfinite timestamps do not create events.

This is the selected request's snapshot, not a complete cross-assignment run
log. The original protocol does not have a separate activity endpoint or an
activity cursor. Type filtering and 20-entry pages are local views of the
captured snapshot. Only explicit Refresh rereads the Host. A successful refresh
shows the returned current request revision and resets the local page; a failed
refresh retains the complete previous snapshot.

Select any visible entry to inspect its full recorded fields:

- Revision: original version title, revision, timestamp and complete goal.
- Assignment: work, delegation, run and original member identities, recorded
  phase/status and waiting or blocking information available in the snapshot.
  No assignment brief is invented when the Host does not return one.
- Report: complete summary and exact report ID from the source work record.
- Settlement: original run/delegation identity and recorded stop reason.
- Delivery: complete summary, every evidence/remaining-issue item, applicability
  and source report/settlement/file/page identities.

Source-report buttons call the existing
`store/results {projectId, workstreamId, delegationRef}` route and verify the
captured report ID where present. They show the full original report with all
evidence and its recorded requirement/version. They never substitute the latest
report. Back returns to the same event and then the same timeline filter/page.
Failed rereads retain the previous body for that exact source.

## Identity and safety

Host connection, Project, request, snapshot revision, reader instance, selected
event and source-reference identities fence every operation. UI callbacks also
carry the captured view generation, preventing old page actions from changing
a newer selection. Request-navigation changes, close and collapse cancel the
owned request and reject late responses. Reads of the shared client use dshMu
before the project-state mutex.

No write, retry, delegation, model or tool endpoint is added. File/page values
remain literal provenance; this view does not open Host paths on the desktop,
execute documents or navigate external links. PB-P11's cross-assignment process
location remains separate.

## Original evidence

Pinned to ChisaAlter/WhaleIsle at
`e9f584be6d3b491ce777bb6823f30bc2bbc6f195`:

- `vendor/dsh-project/client/client.js`, blob
  `412aac7679f65541b60faaddeac69ef135bf97c4`, lines 474–512: request tabs;
  504–512 locally build/sort the Activity rows. Source-report actions are in the
  neighboring results/member views, not a standalone event-log API.
- `vendor/dsh-project/lib/requirements.js`, blob
  `a9f499f634d718b2c8a4ae9196d1246d92da891a`, lines 354–376: work/run/report
  identities and timestamps; 413–423: complete delivery and version DTOs;
  443–446: exact requests/get card source.
- `vendor/dsh-project/lib/service.js`, blob
  `83ae9045a68a8b222ae6d97572a3eb6e4f701f06`, lines 951–957 and 1032–1044:
  existing exact recorded-report and request routes.

One new in-memory fixture ran once with `-count=1`. No real Host, Project data,
files, models, tools, existing suites, race suite or actual GUI was accessed.
The cumulative integrator owns Windows main compilation.
