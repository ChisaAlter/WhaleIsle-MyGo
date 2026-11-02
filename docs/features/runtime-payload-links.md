# Closed native runtime dependency links

The installed rc.2 vendor graph contains cyclic pnpm/workspace dependencies.
Windows release staging inventories physical files without walking aliases,
records SHA-256 hashes and vendor-relative link targets in `runtime-graph.json`,
and emits an exact NSIS file include. A recursive NSIS `File /r` is rejected for
this profile. The original inputs and existing dist output are retained.

The installer embeds its newly built `runtime-links.exe` in `$PLUGINSDIR` and
never executes a previous installation's helper. This windowsgui helper uses
Win32 junction APIs directly, with no command shell or Developer Mode privilege.
Directory aliases must be mount-point reparse tags; file aliases are same-file
hardlinks. Every ancestor and physical target is checked, unknown reparse points
are rejected, and only declared internal aliases may be created or removed.

The ownership manifest keeps its original F/D contract. It is written before
links exist, so no dependency graph can cause recursive ownership traversal.
Promotion restores aliases at their final absolute location. Before upgrading,
a strict old-tree verification rejects missing aliases, changed files or extra
code before anything is detached. Old aliases are detached before the directory
rename and restored if promotion or registration fails. New-tree rollback and
uninstall retry use graph-only detachment without requiring every recorded file
hash. Only when an alias is already absent, detachment permits removed physical
targets and parent directories after checking every still-existing ancestor for
raw reparse points. A still-present alias retains all physical-target, tag and
mapping checks. Verification, restoration and upgrade checks remain strict.
A failed detachment compensates earlier successful removals; a failed restoration
removes aliases created by that call.
All removals unlink the alias itself, never recursively follow its target.

Validation on Windows used seven actual helper fixtures, including a locked
second junction to exercise compensation, an AF_UNIX reparse point with an
unknown tag, cyclic directory aliases, hardlink aliases, stale absolute targets,
missing aliases, changed/external targets and modified old physical files.
`scripts/nsis-runtime-links-fixture.ps1` compiled and ran its own minimal silent
installers under a fresh temporary root: install, forced post-promotion rollback,
upgrade, unknown-junction refusal, then uninstall. It asserts payload bytes remain
on refused removal and that installed/retired owned directories disappear after
successful removal. It uses an isolated application identity and never writes
product registration, shortcuts, production profile or the installed application.

Evidence from this session is in `tmp/release-chain-closeout-3-utf8.log` for the
seven helper and Go resource tests, and
`tmp/release-chain-final-fixtures-utf8.log` for JavaScript graph/resource fixtures
and the original actual NSIS transaction. The later
`tmp/release-chain-retry-check-3-utf8.log` records all ten helper tests and the
actual partial-cleanup retry: an unowned sentinel causes real cleanup to retain
a management tree after deleting owned runtime targets; its bytes survive, the
runtime graph is genuinely gone, and the second NSIS retry safely skips that
absent graph and removes the retired root after only the fixture sentinel is
removed. Three additional helper tests separately cover a still-present graph
with absent aliases/targets/parents, planted reparse ancestors, and a present
alias whose target is absent. The earlier closeout log retains an older
synthetic mobile fixture failure; the final fixture log records its corrected
successful run. This is a minimal transaction fixture, not certification of the
entire product installer UI or all system permission/signing scenarios.
