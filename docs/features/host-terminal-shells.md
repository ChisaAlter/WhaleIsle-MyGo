# Retained Host shell selection and client-local memory

The Host terminal panel can explicitly discover the selected Session's available
shells and choose one by its verified path. Choosing or remembering a shell does
not allocate, retain, attach or run anything. Only New Host terminal permits
allocation; existing retained terminals always keep their existing shell.
The current Host is queried again before allocation. A remembered unavailable
path falls back to the Host's first offered profile. An explicitly selected
profile that disappears fails visibly rather than silently launching another.
An empty successful discovery omits shellPath and uses the Host default, as the
original client does. Discovery failure never falls through to allocation.

## Exact original chain

Pinned source: `ChisaAlter/WhaleIsle@90465b67ec16765631d54c195af19d9ff3c3a770`.
Paths are relative to `vendor/deepseek-harness/`:

- `packages/api/terminal-controller/src/index.ts`, `shells(agent, signal)`:
  discovers verified execution-provider profiles, configured/system default
  first. Real wire call: `terminal/shells` with `{agentId: session}`.
- The same file's `spawn`: `terminal/create` takes
  `{agentId: session, request: {id, cols, rows, shellPath?}}`. The Host resolves
  an omitted path or verifies an explicit path against its discovered profiles.
  Shell arguments remain Host-owned. No client-provided shell command exists.
- `packages/api/terminal-controller/src/client/index.ts`, `launchShells` and
  `selectShell`: remembered available path or first profile, local-only memory.
- `packages/api/terminal-controller/src/client/model.ts`, `refresh/create`:
  only a new terminal chooses a shell; restored terminals adopt their own info.
- `packages/api/terminal-controller/src/client/shell-preference.ts`: browser
  localStorage key `dsh.terminal.shell`, with read/write failure nonfatal.

Verified pinned Git blob IDs:
- Controller index: `24bf1a1f01179a807cff541c2a9e9ffde7cd6bd5`
- Client index: `8ae538dc8104e6a2ffdad2b38bdef516316c17a9`
- Client model: `64bea4906edb40abe413178d80140802d2045422`
- Shell preference: `8e8a4615914475fbd9a51d84f211fe91411a041a`

## Native adaptation and ownership

Native persistence uses `config.json` key `goHostTerminalShell`, in the existing
client data directory, via the existing atomic Config.Update API. This preserves
the original client-local, cross-Session preference semantics without inventing
a Host settings RPC. It does not import browser localStorage or claim shared
browser/native preference storage. Write failures keep the in-memory selection,
show a warning, and do not block an explicit launch. Unrelated config keys survive.
The selector shows only Host-offered paths and their names, not a free-form
executable or command field. Unlike the original guide's click-to-open-tab path,
native selection and New are deliberately separate explicit actions.

Discovery is canceled on hiding/session navigation or transport change, and
late responses cannot cross the Host epoch. Creation captures the Session and
choice at the click, revalidates before the create RPC, and cancels before
allocation if the view has hidden or Host readiness has changed. A failed or
uncertain create is never automatically retried; existing identity/close rules
continue to apply. Preference saves run outside Store/main UI state mutexes.

## Verification boundary

Nine new synthetic fixtures run once, `-run '^TestHostTerminalShells' -count=1`:
1. Discovery/selection wire identity, snapshot isolation and zero allocation.
2. Unavailable remembered path fallback and exact create shape.
3. Disappeared explicit choice preserves selection and prevents create.
4. Persistence failure still permits explicit create.
5. Replaced Host rejects stale discovery and selection.
6. Hide cancels discovery and pre-allocation work.
7. Discovery failure preserves selection and has no automatic retry.
8. Empty discovery omits shellPath for the Host default.
9. Real local config round-trip preserves the preference and unknown keys.

All Host responses and streams were in memory; the config round-trip used a
fresh temporary directory. No old T05 or rename fixtures, race tests, live Host,
shell, native DLL or GUI were run. The integrator owns Windows/UI compilation.
Multiple occurrences, saved navigation, selection-to-chat, application URL/file
routing and read-only terminal selection parity remain outside this increment.
