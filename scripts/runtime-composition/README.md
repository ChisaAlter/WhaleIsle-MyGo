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

## Recorded decisions

### `apply_patch` stays off

The 0.2.2 line added an `apply_patch` tool and defaults it off:
`applyPatch: z.boolean().default(false)` in the `tool-fs` config schema
(`packages/fs/tool-fs/lib/index.js:2012-2017`), and the composition row only
restates that default. The desktop's own users therefore do not have the tool,
and enabling it here would make MyGo the only Host on this surface that differs
from the one we are matching. Do not "fix" the row to `true` without a product decision to
diverge; if that decision is ever taken, it belongs in this section with the
reason.

### `fs-observation-policy` stays disabled

Upstream 0.2.2 disables the policy in its own base bundle; 0.2.1 mounted it. This
is a real behaviour change, not a mount-order detail, and it is inherited by every
Host on this composition, the desktop's included:

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
  ... without reading it first" error (`FS_NOT_OBSERVED`) is unreachable because
  it sits inside the `createIfAbsent` branch. An edit still requires the literal
  `old_string` to match, so a concurrent change is caught only when the text no
  longer matches; write is the path with no check at all.
- `tool-fs` offers no replacement option: its `Config` has tool toggles and read
  limits only (`packages/fs/tool-fs/lib/index.js:2012-2022`).

So this is a genuine loss of edit safety relative to 0.2.1, and keeping parity
with the desktop means keeping it. MyGo can re-enable the row in its own staging
if the product decides to; this section is the place that records that decision
with its evidence, so the next reader does not rediscover it as a defect or
"fix" it by accident.
