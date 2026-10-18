# Scoped audio/video preview and floating player

This is a bounded replacement for WhaleIsle `e9f584be6d3b491ce777bb6823f30bc2bbc6f195`
`src/main/preview-file-window.js` audio/video kinds and the original local
`preview-workspace.js` media URLs. The original Electron floating window used
local `fs` reads, not a remote streaming transport.

## Existing readable contract

The current generated Go API already registers `workspaceFiles/stat` and
`workspaceFiles/readBytes` with `workspaceFileScopeId`, `path`, and `options`.
Upstream `packages/api/workspace-files/src/index.ts` resolves the explicit Session
header and delegates to its composed filesystem. `readBytes` returns
`absolutePath`, `version`, `bytes`, `offset`, `eof`, and native bytes. Existing
`internal/dsh/multipart.go` assembles the binary part as base64 JSON, so a typed
Go `[]byte` receives those bytes without a new RPC or new transfer protocol.

Generic RPC responses remain capped at 8 MiB. Media requests use existing
1 MiB byte ranges and a 32 MiB total limit, with stat before and after. Every
range must match the initiating opaque Host path, version, total size, exact
offset, expected length and EOF. Missing size/version, any changing response,
backend error, replacement request or stale Host/Session epoch aborts. Host
paths are never passed to a client-side filesystem API. The reader's authority
is the original session filesystem permission, not access inferred from a
local path with the same spelling.

There is no reusable scoped document store in this tree. The old generic
attachment cache can look up by attachmentId alone and is therefore deliberately
not used for media authority. Existing PDF/HTML document leases supply the
lifetime and native isolation pattern; F10 borrows the loaded media lease.

Remote support is conditional on the configured Host filesystem actually
implementing these published byte/stat methods. No SSH connection, remote
streaming endpoint, download-to-local path, fallback Host, or attachment reader
is invented. A Host without the methods displays its error. This does not claim
all remote audio/video or an unbounded streaming service is supported.

## User flow and lifecycle

Files offers Media preview and Floating preview for recognized audio/video
extensions. Binary media is never decoded as text or offered for editing;
pre-existing dirty text drafts remain visible and unchanged. Explicit media
file-card clicks use the same Host reader before local path resolution.
Nothing auto-plays. The fixed audio/video element uses native controls, including
seek, and codec support remains dependent on the installed WebView2 runtime.
The page and toolbar state the 32 MiB cap, 30-minute lifetime and codec limit.
If playback fails, the existing Files external-open action is a separate click.

The read captures the exact client, selected Session and selection generation;
the capability additionally checks main-window ownership, Host generation,
endpoint/cookie identity and Session cwd. Old reads cannot publish after a
replacement open. Selection away-and-back cannot revive a prior lease. Closing
the source tab/window, owner invalidation, application exit or 30-minute expiry
revokes bytes and closes the native player. Floating windows borrow the source
snapshot, do not reread it, and closing only the floater keeps the source lease.

## Type and resource boundary

Recognized extensions mirror the original audio/video sets: AAC, FLAC, M4A,
MP3, OGA/OGG, WAV; M4V, MOV, MP4, OGV, WebM. The reader checks matching container
signatures and bounded framing (RIFF/WAVE chunks, FLAC STREAMINFO, MPEG/ADTS
headers, Ogg identification packets, WebM DocType, or ISO BMFF ftyp/moov/mdat).
These are container/type gates, not full codecs, guarantees that every valid
media file is accepted, or proof of the installed decoder's allocation limits.
Playlists, URLs, unsupported containers, mismatched extensions and oversized
files are rejected. No codec/software is installed.

A private random loopback port and 256-bit capability expose exactly
`/<token>/player.html` and `/<token>/media`, GET/HEAD only. The document consists
only of fixed markup and an escaped filename. It has `script-src 'none'`,
`connect-src 'none'`, and `media-src` permitting that one media URL. Existing
HTML `childCSP` and `DocumentCSP` remain untouched. Responses are no-store,
no-referrer, nosniff, deny framing; Host, path, query, Origin and cross-site
requests are checked. No directory service, arbitrary file/URL parameter,
Host cookie forwarding, upload or external network proxy exists.

Media implements one byte Range with correct 206, Content-Range, Content-Length,
MIME and HEAD semantics. Multipart, malformed and out-of-bounds explicit ranges
are rejected with 416. Reads serve immutable captured bytes only.

`LocalMediaURL` is a separate, mutually exclusive native preview option.
Windows uses the existing temporary IsolatedPreview WebView profile, disables
scripts, native bindings, preloads, WebMessage, Host objects, developer tools,
dialogs, context menus, downloads, permissions and popups. Top navigation only
admits the exact player URL; resource interception only admits GET/HEAD for
that player or its one derived media URL. No child frame navigation is admitted.
Browser inspection, script-based pick/capture, tools and Browser profile actions
exclude media tabs. Native media rendering is the only decoder; its runtime
compatibility and Windows visual behavior have not been accepted by this work.

## Verification

One synthetic fixture pass (`go test -count=1 ./...`) used a 48-byte WAV and an
in-memory fake caller plus httptest request/response recorders. It covered scoped
stat/read/stat, version mismatch, path rejection, container mismatch, byte cap,
immutable snapshot, fixed no-script shell/CSP, HTTP Range/HEAD/method/Host/query/
Origin/expiry, and exact native capability URL policy. No HTTP client request,
real Host, user file, media decoder or playback was used. A loopback listener was
created only for the synthetic capability. Checks are retained with the handoff,
not added as a permanent legacy regression suite. No repeated fixture or race
run was made. Initial compile-only attempts failed before fixture execution;
the one actual fixture execution passed. Integration owns the cumulative Windows
compile of the exact final source and new session/main wiring.
