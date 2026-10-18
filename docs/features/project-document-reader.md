# Native Project document reader

Projects → Project documents opens the selected Project's `docs` inventory from
its authenticated Host. Select a listed file to read its entire current text in
a native, selectable, read-only text area. Reload rereads that same document;
Back restores the inventory, and closing or collapsing Projects cancels a
pending document request. A failed reread retains the last successfully read
body with an explicit error. An initial failed read does not display another
file's contents.

The original `dsh-project/store/list` and `dsh-project/store/read` protocols are
used, with the original 512000-byte read limit. This is a current text reader,
not a historical snapshot, document editor, binary viewer or external opener.
Host paths are never interpreted as paths on the desktop. Text, including HTML,
Markdown links, script-looking contents and file paths, stays literal. Files
outside the readable docs scope or beyond the text limit are disabled with an
explanation. Missing body data is distinguished from an explicitly empty file.

## Identity and lifecycle

The document inventory belongs to the selected Project and authenticated Host
connection. Reads must select an exact inventory entry. A response must return
the requested source path. Host replacement, project replacement, new reads,
Back and close fence out stale results. Closing an old request cannot clear a
new request's busy state. Document text is not persisted to a local file.

## Parity scope

This completes the standalone docs inventory-to-body path in PB-P09. It does not
complete PB-P09's request-authority or grouped historical-delivery collection,
PB-P07's activity timeline or PB-P11's exact historical process location.
Existing notes/preferences, recorded reports and their validated artifact opens
remain separate. The original full Project plugin UI remains available through
the Host WebView when supplied and enabled.

## Protocol evidence

Pinned to ChisaAlter/WhaleIsle at
`e9f584be6d3b491ce777bb6823f30bc2bbc6f195`:

- `vendor/dsh-project/lib/service.js`, blob
  `83ae9045a68a8b222ae6d97572a3eb6e4f701f06`, lines 863–876: Project root,
  readable docs scope, symlink/realpath checks and text byte limit; lines
  933–949: docs inventory; 1042–1044: public read-only control-plane routes.
- `vendor/dsh-project/client/client.js`, blob
  `412aac7679f65541b60faaddeac69ef135bf97c4`, lines 920–959: original materials
  navigation and standalone docs entries. The native reader uses store/read
  to show the body directly rather than handing a Host path to a desktop opener.

Verification uses only new, in-memory HTTP fixtures. No real Project, Host,
model, tool, user document or GUI was started or read. Windows main compilation
is performed by the cumulative integrator.
