# Git branch action contracts

Original reference: WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`,
`src/main/git.js` and the vendored `client/ui-git` README and BranchMenu.

- Pull now passes `--ff-only`, matching the original user action. Divergence
  produces the Git error instead of creating an unexpected merge/rebase.
- The native Git workbench offers a filterable local/remote branch list,
  Switch, Refresh branches, and Create and switch.
- Remote symbolic HEAD rows are hidden. Current branch comes first; redundant
  origin rows with a local match are omitted.
- Switching a remote-tracking branch creates a local tracking branch rather
  than detaching HEAD. Inputs pass `git check-ref-format --branch`; selection
  is checked against the current list before execution.
- Normal Git dirty-tree and other-worktree protections remain enabled. There
  is no force, reset, stash, implicit push or branch replacement.
- Branch changes clear the staged-review token and refresh the workbench.
  Existing dirty editor tabs and save-conflict protection remain intact.
- Errors remain visible in the branch controls, distinct from an empty list.

Validation: source/argument and binding checks only. No Git command, workspace
mutation, network Git request or model was executed. Cloud disk exhaustion
prevented compiler/runtime verification.
