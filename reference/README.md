# Reference snapshot

`electron/` is a read-only copy of the Electron app at
`C:\AI\Deepseek-Harness-Desktop`, used as the behavior reference for the
MyGo rewrite. Edit the original project, not this copy.

- Branch: `codex/plugin-startup-guidance`
- Commit: `26205a89fb7a6d416ef446d6281cb77afe610bcd` (2026-10-06 17:03 +0800)
- Snapshot taken: 2026-10-09
- Contents: files tracked by git plus untracked files that are not ignored,
  including 22 uncommitted changes in the working tree at that time.
  Ignored output (`node_modules`, `dist`, `.tmp`, `.worktrees`, `build`
  artifacts) is not included.

The original keeps changing. To see what changed after this snapshot, run in
the original repository:

```powershell
git diff 26205a89fb7a6d416ef446d6281cb77afe610bcd --stat
```

Uncommitted changes that were in the snapshot show up in that diff too.
