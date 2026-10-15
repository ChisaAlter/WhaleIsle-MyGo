# Restore desktop pet appearance and behavior

Settings → Pet now offers **恢复外观与行为默认 / Reset appearance and behavior**.
An explicit confirmation lists the eleven appearance/interaction defaults and clears
the pet-specific vision provider/model pair, turning Look off.
Cancel and dialog failure do not save; a save failure keeps the current editing
draft. Success saves immediately and updates the existing pet through its current
behavior/status path without recreating the window. A changed personality uses
the existing personality synchronization callback.

Defaults are scale 1, opacity 1, natural personality, balanced activity; self-talk,
wandering and chat enabled; position lock, Shift-to-drag, power saving and click
sound disabled. The action starts from the committed configuration, so it cannot
save unrelated pending visibility, skin or THA4 edits. Only the behavior draft is
replaced after a successful conflict-checked atomic save.

## Original contract

Verified against original commit `e9f584be6d3b491ce777bb6823f30bc2bbc6f195`:

- [PetSection.tsx:433–445](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/deepseek-harness/packages/client/ui-settings-general/src/client/PetSection.tsx#L433-L445)
  wires the actual reset button to `save({reset:true})`.
- [desktop-live2d.js:546–558](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/desktop-live2d.js#L546-L558)
  replaces only `state.settings`, persists, pushes the result, and mirrors a
  changed personality. [Lines 62–76](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/desktop-live2d.js#L62-L76)
  show visibility, coordinates, growth, stats, activity watermarks, file history
  and assistant session identity outside `settings`; reset preserves them.
- [pet-settings.js:26–43](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/pet-settings.js#L26-L43)
  supplies the exact defaults. The eleven native behavior fields agree.

The subsequent [pet-specific Look model increment](pet-look-model.md) adds and resets
`lookModel` and `lookProvider`. The original `approvalButtons` is a reserved
normalized field with no available approval bridge, not a missing shipped approval
feature; see [the source boundary](pet-approval-reserved.md).
Global Host vision configuration is unchanged. Skin and THA4 are native-only
selection fields outside behavior.
The legacy Electron state, care/growth ledger and saved native position survive.

## Verification boundary

One new synthetic Go fixture checks all eleven defaults, persistence/reopen,
legacy and unrelated values, unrelated UI drafts, changed-personality callback,
repeat reset and stale-write/unavailable-store failures. It uses temporary local
configuration only. Window-bound methods are stubbed in that standalone run;
actual dialog appearance, renderer update and native window behavior are not run.
The integration owner performs the single cumulative Windows main build.
No real Host, model, desktop, credentials or user settings are used.
