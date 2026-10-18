# Native historical delivery collection and report evidence

Projects → Delivery history reads recorded handovers with the original
`dsh-project/store/list` payload `{projectId, kind: "deliveries", cursor?}`.
The collection groups the current page by request identity and shows every
returned delivery identity and request revision, including multiple handovers
at the same revision. Older/Previous follows the Host's opaque delivery cursor;
failed reads keep the previous page and do not advance its cursor stack.

Select a delivery to read its complete recorded summary, every evidence item,
remaining issues, timestamp, outcome, applicability and historical status.
Historical/obsolete delivery is not shown as proof that the current request is
complete. Back preserves the collection page.

Each source report is a real navigation action. It reads
`store/results {projectId, workstreamId, delegationRef}` and verifies the exact
recorded delegation and report ID when supplied by the original handover. It
shows the full report, all evidence/remaining issues and its original request
provenance. A source report can predate the handover's request revision; its
recorded version is preserved, rather than overwritten with the latest version.
A changed/missing report fails explicitly without substituting the latest report.

## Boundaries

- The authenticated Host, Project, delivery ID, request ID/revision, cursor
  chain and exact report reference are scoped to one reader instance.
- Back/close/newer requests invalidate late results; canceling an old request
  cannot clear the busy state of a newer one. Closing or collapsing Projects
  cancels the owned request. A stale frame cannot open another Host's history.
- All data reads are read-only. No retry, delegation, model or tool RPC is added.
- Artifact, settlement and published-page identities are retained as text in
  this view. It does not treat Host paths as desktop paths or execute pages.
  Existing separate report-file routes and conditional full-plugin access are
  unchanged.

This closes the historical-delivery collection → full body → exact report
body/evidence navigation portion of PB-P09. It does not complete the
requirement-authority error collection, published-page rendering, new artifact
openers, PB-P07's activity timeline or PB-P11's historical process location.

## Original source

Pinned to ChisaAlter/WhaleIsle at
`e9f584be6d3b491ce777bb6823f30bc2bbc6f195`:

- `vendor/dsh-project/lib/requirements.js`, blob
  `a9f499f634d718b2c8a4ae9196d1246d92da891a`, lines 173–184: exact original
  report/terminal references; 207–215: complete handover record fields;
  322–330: recorded status filtering, Project-bound delivery cursor, full
  handovers and artifact identities.
- `vendor/dsh-project/lib/service.js`, blob
  `83ae9045a68a8b222ae6d97572a3eb6e4f701f06`, lines 933–934 and 951–957:
  original delivery-list and exact-report routes.
- `vendor/dsh-project/client/client.js`, blob
  `412aac7679f65541b60faaddeac69ef135bf97c4`, lines 804–838 and 880–883:
  full delivery details, exact report buttons, per-request revision grouping
  and original cursor payload.

Verification: one new in-memory fixture, once with `-count=1`; no real Host,
project data, file, model, tool, existing suite, race suite or GUI was run.
Windows main compilation is delegated to the cumulative integrator.
