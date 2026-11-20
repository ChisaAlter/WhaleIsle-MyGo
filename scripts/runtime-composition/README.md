# Staged Host composition contract

`runtime*/` is gitignored and every staging path copies a source graph verbatim,
so the mounted rows a Host runs with depend on the source a stage came from. The
desktop app vendors its own rows on top of the upstream pin (`mcp-servers-file`,
`llm-vision-fallback`, its DeepSeek tools plugin), which means a stage built from
a bare upstream snapshot silently produces a Host that is missing rows MyGo's own
code depends on. This directory is the record of what a stage must carry, per
harness version, and `scripts/session-runtime-composition.mjs` is what enforces
it after staging and before packaging.

`contract.json` holds, for each supported version, the sha256 of
`packages/bundle/base/cordis.patch.yml` that the matching desktop release
vendors, the commit that owns that file, the rows and config the desktop adds
over the pin, and every recorded variant. `upstream/` holds the pinned upstream
compositions, hash-verified against the contract, so a failure can name the rows
a stage lacks without a desktop checkout or network access.

The composition a start actually runs is that file plus MyGo's own `--patch`
overlays under `internal/harness`. The contract covers the staged file; the one
deliberate departure from the desktop composition lives in an overlay and is
recorded under Recorded decisions.

## Checking a stage

```
node scripts/session-runtime-composition.mjs verify <runtime-root | vendor-dir | harness-root>
node scripts/session-runtime-composition.mjs diff   <tree> <tree>
```

`verify` exits non-zero and names the problem: rows the desktop composes that the
stage lacks, rows still under their upstream id, rows the contract cannot account
for, missing desktop config, CRLF line endings, or a stage whose pin record names
another harness pin. `diff` reports two trees' row-level differences, which is the
useful reading when two versions are involved: the versions it prints decide
whether a difference is drift or a gap.

`scripts/session-runtime-composition.test.mjs` covers the checker itself and
re-verifies that `upstream/` still matches every recorded pin hash. It runs in the
Windows packaging CI step.

## Updating the contract

A harness pin bump, or a desktop release that changes its composition, fails
`verify` until the contract is updated. That failure is the review.

1. Stage the runtime from the desktop source that vendors the new harness version
   (`build-harness.ps1 -RuntimeSourceResources <installed desktop>` or
   `-SourceMode current`).
2. Copy the new upstream composition for the pin into `upstream/` and record its
   sha256.
3. Record the desktop's file sha256, the desktop commit that owns it, and the rows
   and config it adds over the pin. When the desktop has shipped the same harness
   in two states (before and after one of its own plugins), record the older file
   as an `alternates` entry with the reason, instead of loosening the check.
4. Run the checker against every stage on the machine and the desktop checkout.

## Bumping the harness from the source fork

Every staging path above starts from a tree 本体 or the desktop already built and
copied. `scripts/harness-fork/` replaces that with a recorded git history MyGo
builds from, so a harness bump is one merge and one build instead of a restage of
compiled artifacts.

`scripts/harness-fork/pin.json` is the record: each upstream tag as a commit on
`refs/harness/upstream/<ref>`, 本体's vendored tree as one commit on
`refs/harness/carry`, and every bump as one merge whose human resolution lives in
`scripts/harness-fork/resolution/<ref>.patch`. Every tree hash in the pin is
checked while the record replays, so a tarball that no longer matches its tag, or
a resolution that no longer applies, fails before a build starts. `fork.mjs
materialize` replays the record into `.harness-fork/tree`; `stage.mjs` builds,
stages and gates a runtime from it; `build-harness.ps1 -SourceMode fork` runs
both. Desktop plugins and the mobile bridge are not the fork's, so that mode
needs `-DesktopResources <installed desktop>`.

### The bump

1. **Merge the upstream tag.** `node scripts/harness-fork/fork.mjs bump <ref>`
   downloads the tag, imports it as a commit and merges it into the fork. It
   resolves mechanical conflicts by policy — `pnpm-lock.yaml` is never merged, it
   is regenerated — and leaves the rest in the worktree with a list. Conflicts in
   `packages/client/ui-*` are 本体's slot API against upstream's, and each one is
   a product decision rather than a merge trick. If this host routes HTTPS
   through a proxy that cannot reach codeload, this first step fails and names
   that proxy: pass `--no-proxy` (through `build-harness.ps1`, `-NoProxy`) to
   download directly. Nothing bypasses the proxy on its own.
2. **Resolve the conflicts.** Never resolve by dropping a package 本体 carries:
   `runtime-fork-audit.mjs --expect=pin.json` fails the stage when the carry no
   longer matches the pin's package list, which is the audit list this contract's
   `staging.desktopPackages` is a subset of. A resolution that deletes a fork
   package is a decision to stop shipping that feature, not a merge.
3. **Rebuild and record.** `fork.mjs record <ref> --why "<the decision>"` stores
   the resolution patch and the tree it produced, then `build-harness.ps1
   -SourceMode fork -HarnessForkTag <ref>` replays the record, builds, stages and
   runs the gates. A textually clean merge is not a clean build: upstream's new
   packages can reference types 本体's contract changed, and each such seam is a
   one-line adapter in the same resolution patch. Budget for that — rc.5 needed
   two, both surfaced by upstream's new `deepseek-chat-import`: a required owner
   prop relaxed to optional in 本体's `ui-conversation` contract, and a type
   export 本体 had renamed in `ui-workspace`.
4. **Record the contract entry only when the bump ships a release.** A bump onto
   a tag 本体 already vendors needs no new entry: the contract already names that
   version and `verify` passing is the proof. A bump to a version no 本体 release
   vendors needs a derived entry, recorded as "Updating the contract" says,
   together with the restage that ships it. Pre-recording an entry for a version
   that is not shipping only makes the next bump's failure ambiguous.

The bump's cost is therefore one merge plus one build, and the parts that need a
person are the slot-API conflicts and the seams the build finds. Everything else
— the tag's tree, the carry, the resolution — is pinned, so a later bump starts
from the tree that was actually reviewed rather than from whatever the desktop
last installed. The desktop plugins the stage carries are the exception the bump
cannot merge: their dsh compatibility declarations are 本体's, and the stage
widens them for the version it ships. That widening is recorded, not silent —
see "Carried plugin compatibility is widened at the stage".

## Recorded decisions

### `apply_patch` stays off

The 0.2.2 line added an `apply_patch` tool and defaults it off:
`applyPatch: z.boolean().default(false)` in the `tool-fs` config schema
(`packages/fs/tool-fs/lib/index.js:2012-2017`), and the composition row only
restates that default. The desktop's own users therefore do not have the tool,
and enabling it here would make MyGo the only Host on this surface that differs
from the one we are matching. Do not "fix" the row to `true` without a product
decision to diverge; if that decision is ever taken, it belongs in this section
with the reason.

### `fs-observation-policy` is mounted back

Upstream 0.2.2 disables the policy in its own base bundle; 0.2.1 mounted it. The
staged composition stays byte-identical to the desktop's, and MyGo mounts the row
again in a `--patch` overlay every start passes: `internal/harness/extra_overlay.go`
writes `desktop-host-safety.patch.yml` into the profile and the launcher appends
it to the patch list, so the effective composition is the desktop's file plus
this one row. That overlay is the only place MyGo deliberately departs from the
desktop composition, and it exists because the guard is real:

- The policy is the only thing that supplies the guard. It registers the
  `fs/write-intent`, `fs/edit-intent` and `fs/delete-intent` listeners over a
  per-session observed-file map (`packages/fs/fs-observation-policy/lib/index.js:92-101`).
  Its own header states the consequence, at `:7`: "Without this plugin, tools
  retain the bare provider's unconditional mutation behavior."
- Nothing else supplies those intents. `packages/deliverables/workspace-changes`
  listens only to call `next()` and record the path
  (`lib/index.js:1076-1097`), `liveshare` and `tool-str-replace-editor` ask with
  the same `undefined` default, and `tool-cordis` only publishes their
  signatures.
- The provider enforces only what it is handed. `fs-local.writeTarget`
  (`:1006`) checks `expected?.kind === "replaceIfVersion"` (`:1009`) before a
  write, `editText` checks `if (expected && ...)` (`:926`) and `requireFile`
  does the same for patch, delete and move (`:1019`). With no intent, a write
  overwrites the file on disk unconditionally, and the "cannot overwrite existing
  ... without reading it first" error is unreachable because it sits inside the
  `createIfAbsent` branch. An edit still requires its literal `old_string` to
  match, so a concurrent change is caught only when the text moved; write is the
  path with no check at all.
- `tool-fs` offers no replacement option: its `Config` has tool toggles and read
  limits only (`packages/fs/tool-fs/lib/index.js:2012-2022`).
- Composed profiles show it directly: the 0.2.1-rc.2 profile mounts the row, the
  0.2.2-rc.2 profile carries `disabled: true`, and with the overlay the effective
  row is `disabled: false` again.

Proof, from a real Host rather than a unit test: `tools/core-host-check
-prompt-flow` starts the staged Host against its owned loopback model in an
isolated profile and drives prompts through the ordinary session path. Two checks
cover the guard and both pass:

- `write-without-read-refused` — a `write` to `qa-unread.txt`, a file the session
  never read, is answered `cannot modify "…": file has not been read — read the
  file, then retry`.
- `stale-edit-refused` — an `edit` computed from a read that the file has since
  outrun is refused.

Flip the overlay row to `disabled: true` and the same flow stops at `Host
accepted a write call for a file the session never read`. The checks are not
vacuous: that failure is what a Host on the plain desktop composition does.

Do not remove the row, and do not reduce it to parity with the desktop: write is
the unprotected path, and losing it costs a user their file.

### A version ahead of the desktop is recorded as derived

MyGo stages a 本体 graph verbatim, so a harness version 本体 has not synced has no
desktop file to hash. Its entry still names the pin and the rows 本体 adds, and
`desktop.reference` starts with `derived from`: the previous entry's row deltas
applied to the new pin's upstream file. That derivation is a rule rather than a
guess only because applying it to the previous pin reproduces the previous entry's
recorded sha256 byte for byte, which is the check to run before recording a new
one. Re-record the entry with the restage that ships the version; until then
`verify` refuses any stage whose file differs from the derived one.

### A derived version names the packages only a 本体 vendor carries

`staging.desktopPackages` is the machine-readable half of a derived entry: the
composition rows mount packages by name, and 本体's vendored harness is a fork,
so a stage that copies the upstream tag alone mounts rows it cannot load. `verify`
refuses such a stage, naming each package it lacks — which is the reminder, not a
comment. The list is deliberately what MyGo's own rows and readiness gate require,
not the whole fork: `node scripts/runtime-fork-audit.mjs <upstream-tree>
<desktop-harness-root>` exits non-zero with every package the vendor carries over
that tag (23 package directories for 本体 main against `dsh-v0.2.2-rc.2`), which
is also the answer to "can this version be staged from upstream at all".

### Carried plugin compatibility is widened at the stage

The desktop plugins are copied from 本体's vendor tree, so each one's
`peerDependencies` declares the dsh versions 本体 supports. The Host reads those
declarations on boot and disables a row whose ranges exclude its own version
(`packages/boot/app-boot/src/plugin-compatibility.ts`), printing the reason into
the Host log:

```
dsh: disabling profile plugin row "dsh-whale": Plugin dsh-whale@0.1.0 is
incompatible with dsh 0.2.2-rc.5: requirements {"@deepseek-ai/dsh-agent":
"^0.1.5-rc.2 || 0.2.1-alpha.1 || … || 0.2.2-rc.2"}
```

A disabled row is not a cosmetic warning. The installer's health gate
(`internal/installhealth`) waits for the Host to reach its inventory, and a Host
that dropped the rows it was told to mount never gets there: the rc.5 payload
failed installation on Ayase with a 90-second gate timeout and a rollback, and
the only place the cause appeared was that Host log line. A stage that carries
these plugins therefore has to make the declarations admit the version it ships.

`CARRIED_COMPATIBILITY` in `scripts/harness-fork/plugin-compat.mjs` is the
record. Each row names one carried plugin and the exact peer requirements the
stage widens; the carry applies them while it copies the plugins into a stage's
vendor directory, so the next bump widens to the new version without anyone
editing a manifest by hand. At `dsh-v0.2.2-rc.5` the table is:

| plugin | requirements widened to `0.2.2-rc.5` |
| --- | --- |
| `dsh-whale@0.1.0` | `dsh-agent`, `dsh-session`, `dsh-settings`, `dsh-system-prompt` |
| `dsh-remote@0.8.21` | `dsh-commands`, `dsh-host-webserver`, `dsh-system-prompt` |
| `dshbot@0.2.0` | `dsh-agent`, `dsh-llm`, `dsh-scope`, `dsh-session`, `dsh-settings`, `dsh-subagent`, `dsh-system-prompt`, `dsh-tools` |

Two guards keep the table honest rather than decorative:

- A carried plugin whose requirement needs widening but has no row stops the
  stage. An unrecorded assertion never reaches a payload, so a bump that adds a
  plugin or a peer fails where the decision is made instead of shipping a row
  the Host will disable.
- A plugin that declares `engines.dsh` is refused rather than widened. The Host
  reads that field alone and ignores the peers, so widening peers would change
  nothing while looking like it did.

What the widening asserts is only what the boot gate proves: the plugin mounts
and the Host reaches its inventory. The evidence for rc.5 is
`whaleisle-integration.exe --check-installation <tree> <json>` reporting
`passed: true` on a stage carrying these plugins — not the table, and not a
green CI run. The `dependencies` block in each manifest is left untouched; the
Host's compatibility check reads `peerDependencies` only, and these packages
resolve from the runtime's own vendor tree rather than from an install.
