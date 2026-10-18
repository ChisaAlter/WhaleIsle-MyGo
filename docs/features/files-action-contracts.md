# Files action contract completion

Original reference: WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`,
`packages/client/ui-files/README.md`, `FileTree.tsx` and `FilePreview.tsx`
under `vendor/deepseek-harness`.

- Selecting an existing file tab now rereads its bytes; the explicit Reload
  action uses the same path.
- Reloading a dirty tab retains its draft, original save base and editor on
  changed, missing, binary or truncated content. A later Save still uses the
  existing conflict check. Reload never implicitly writes or discards.
- The preview toolbar exposes separate Copy relative path and Copy absolute
  path controls.
- System Open and Reveal choose one executor: the ready selected-session Host,
  otherwise the existing native OS fallback. A Host request is never followed
  by a second local open or an automatic retry of an uncertain request.
- Existing tab recovery, PDF/HTML routing, remote explorer and file-save
  implementation are retained.

Validation: bounded source/anchor checks and read-through of callback routing.
No user files, model calls, Host sessions or OS openers were exercised. Runtime
and compiler checks were unavailable because the cloud workspace is full.
