# workspacefs search tests — root cause and fix (round 219)

Date: 2026-10-25. Scope: `go test ./internal/desktop/workspacefs -count=1` failing in
`parity_test.go:70` (`search truncated: 0`) and `search_match_test.go:24` (every
subsequence query empty, `prefix-only search: []`).

## 1. Reproduction

```
FAIL: TestSearchHasNoIndexOrMatchTruncation   parity_test.go:70: search truncated: 0
FAIL: TestProjectFileSearchSubsequence        query "FP": got [] … "FilePreview.tsx": got [] …
                                              prefix-only search: [], <nil>
```

The same failures appeared in isolation (`-run Test…`), so they were not ordering
dependent. `TestVisibleDotfilesAndGitIgnoreParity` (which `git init`s its fixture)
passed throughout.

## 2. Minimal fixture and the resolved backend

A scratch program against a plain `os.MkdirTemp` directory returned the files
correctly (`entries=[a.txt b.txt]`, `search a -> [a.txt]`), so the walker, matcher
and `git check-ignore` fallback were healthy. A debug test inside the package
showed the difference:

```
root=C:\AI\WhaleIsle-MyGo\.toolchain\gotmp\TestZzDebugListing601764269\001
entries=[] err=<nil>          # os.ReadDir(root) = 2 entries
```

`t.TempDir()` followed `GOTMPDIR`, which `scripts/env.ps1` sets to
`<repo>/.toolchain/gotmp`. `.toolchain/` is git-ignored (`.gitignore:2`), so the
fixture lives **inside this repository's worktree, under an ignored directory**.
`ListWorkspaceDir` filters entries through
`git -C <root> check-ignore --no-index --stdin -z` (`list.go:66-91`); git resolves
the enclosing repository and reports every fixture path as ignored, so the
directory lists empty and both search walks find nothing.

Backend: local filesystem walk + `git check-ignore` (no ripgrep involved). The
root cause was the fixture location, not the matcher: a workspace rooted inside an
ignored directory of an enclosing repository legitimately hides its files, which is
also what the native workspace does.

## 3. Fix

Test fixtures that model a standalone workspace no longer inherit the enclosing
repository:

- `internal/desktop/workspacefs/fixture_root_test.go` (new): `gitIsolatedRoot(t)`
  wraps `t.TempDir()` and sets `GIT_CEILING_DIRECTORIES` to the fixture's parent, so
  repository discovery stops above the fixture (`git -C <root> check-ignore` then
  reports "not a git repository", the documented fallback at `list.go:77-83`, and
  the listing stays unfiltered — exactly the standalone-directory state the tests
  describe).
- `TestProjectFileSearchSubsequence` (`search_match_test.go:12`) and
  `TestSearchHasNoIndexOrMatchTruncation` (`parity_test.go:58`) now build their
  fixtures with `gitIsolatedRoot(t)`.

No parity assertion was weakened: the ignore-parity test still creates a real
repository with `.gitignore` rules and still asserts ignored files are hidden and
kept files/dotfiles are visible.

## 4. Regression test

`TestSearchFixtureIsGitIsolated` pins the contract:

1. a fixture from `gitIsolatedRoot` lists `keep.txt` and
   `src/client/FilePreview.tsx` even though the enclosing tree ignores the temp
   directory;
2. `git -C <root> rev-parse --show-toplevel` reports no enclosing repository from
   that root.

If someone removes the ceiling or moves fixtures back under `.toolchain`, the
first assertion fails; if the environment stops nesting temp inside a repository,
the test still holds (both assertions are then trivially true), so it cannot turn
into a false alarm.

## 5. Verification

| Command | Result |
| --- | --- |
| `go test ./internal/desktop/workspacefs -count=1` | ok 4.061s (was FAIL) |
| `go test ./internal/desktop/workspacefs ./internal/desktop/components ./cmd/whaleisle -count=1` | ok 5.824s / 4.529s / 20.041s |
| `go vet` same packages | clean |

## 6. Residual

- The trap is environmental: any test that assumes a plain directory will fail when
  `GOTMPDIR` points inside an ignored repository path. The helper is the pattern to
  reuse; other packages creating workspace fixtures under `t.TempDir()` and calling
  git-ignore filtering should adopt it if they hit the same symptom.
- `git` remains a runtime requirement of the listing path; when git is missing the
  code already falls back to the unfiltered listing, and that branch is covered by
  the existing parity tests' skip/fallback behaviour.
- No product code changed in this slice.
