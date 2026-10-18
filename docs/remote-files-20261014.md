# Session-bound remote files

Native remote panel now includes an independent current-session file explorer:
- Explicit browse first verifies `/api/dsh-remote/status?sessionId=...` reports `sessionBound:true` and `sessionMode:remote`, then verifies `ls` reports `bound:true`.
- Read, write and download always include the selected session ID. No machine-default fallback or invented sync endpoint.
- Preview uses the original 256 KiB read cap. Binary/truncated/missing-mtime previews cannot be edited or saved.
- Explicit saves send original expectedMtime. Conflict, transport failure or incomplete receipt retains the draft and blocks automatic retry until the user discards/re-reads. Drafts remain separate across session changes.
- Explicit confirmed download uses `fs` op `download`, shows returned byte count and mirror path, and warns it may overwrite a same-named mirror file. It is not bulk synchronization and reports no invented percentage.
- Session/Host changes cancel in-flight requests and invalidate stale responses. An uncertain write marks its retained draft stale.

Source: original legacy `dsh-remote/plugin-src/client/{api.js,RemoteExplorerBody.jsx,RemoteFileBody.jsx}` and `lib/{routes-fs.js,remote-fs.js,index.js}`, available in the current Host staging component. The fixed 90465b67 desktop source contains `chisacode-remote`; this change does not replace the preserved legacy Host component or create a compatibility RPC. A Host without the explicit session-bound status/list contract reports unavailable.

Validation: one mock HTTP fixture for explicit binding, preview, mtime conflict, preserved draft, explicit reread, save and download. No SSH connection, real file read/write/download, Host, model, credentials or actual session was accessed. Windows cumulative compile belongs to integration owner.
