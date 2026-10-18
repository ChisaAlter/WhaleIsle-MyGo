# Git initialize, publish and commit/push actions

Original source is WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`:
`src/main/git.js` (`gitInit`, `gitPublishRepository`, `gitPush`) and
vendored `client/ui-git/PublishDialog.tsx` / `GitActionsControl.tsx`.

## Native user actions

- Non-repository folders expose Initialize Git in the Git workbench. It creates
  an empty local main branch; no implicit commit or push.
- Publish offers an existing HTTPS/SSH URL or a new GitHub repository name and
  explicit visibility. New repositories default to private.
- Existing URL adds origin; a repository with commits then pushes its current
  branch. An empty repository only gains the remote.
- Blank URL uses the already installed/authenticated gh CLI with the original
  repo-create/source/origin contract, adding --push only when commits exist.
- A review page states the destination, visibility and upload of committed
  history. Cancel/Back do not mutate anything; final Confirm starts the action.
- An existing origin fails closed rather than being overwritten.
- Commit and push uses the existing exact staged-diff token. It shows the
  resolved branch, remote, destination ref and URL before final confirmation.
- Top-level Push uses the same destination review and normal non-force push,
  establishing upstream when needed. Default-branch destinations are explicit.
- A changed branch/remote destination invalidates the push review.
- Push failure after a successful commit preserves and reports the local
  commit; users retry Push, never an automatic duplicate commit.
- Workbench operations run off the UI thread, suppress repeated submission,
  keep their originating workspace and report partial failure. They do not call
  a model, auto-stage files, force-push, merge or deploy.

Source verification covered action bindings, literal argv, review gates,
partial-result handling and background completion paths. No real Git command,
remote mutation, gh authentication or publication was executed. Compiler/runtime
verification is blocked by cloud disk exhaustion.
