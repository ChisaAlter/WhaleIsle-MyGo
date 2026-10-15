# Desktop pet approval is reserved upstream

At original commit `e9f584be6d3b491ce777bb6823f30bc2bbc6f195`,
`approvalButtons` is a persisted, normalized boolean with default `false`.
It is not an available desktop-pet approval feature. MyGo must not count the
absence of a pet approval bridge as a missing shipped upstream workflow.

The bounded source check found:

- [pet-settings.js:26–43,69–90,105–110](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/pet-settings.js#L26-L110) defines, normalizes and patches the field; resetting restores `false`.
- [PetSection.tsx](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/deepseek-harness/packages/client/ui-settings-general/src/client/PetSection.tsx) has no approval toggle. Its rendered controls do not expose the reserved flag.
- [src/preload/index.js:279–281](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/preload/index.js#L279-L281) declares `respondApproval` for `shell:live2d-respond` and an alert subscription, but [desktop-live2d.js](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/desktop-live2d.js) registers no responder. Exact repository search for that channel found the preload declaration and documentation, not a main-process implementation.
- [pet-live2d.js:80](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/renderer/pet-live2d.js#L80) only initializes the flag. [Lines 5077–5087](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/renderer/pet-live2d.js#L5077-L5087) accept generic alert metadata, including `buttons`, without an implemented user approval-response path.
- The original [feature boundary at line 67](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/docs/features/desktop-live2d-pet.md#L67) explicitly states that the R3 approval bridge has no current code path. [Line 19](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/docs/features/desktop-live2d-pet.md#L19) says its toggle stays hidden until the bridge exists. Planned allow-once/reject semantics elsewhere in that document do not establish a working consumer.

MyGo's existing ordinary chat approvals are separate. This clarification adds
no pet approval UI, permission request handler, automatic approval behavior or
configuration write. The legacy Electron configuration remains preserved by the
existing raw-config mechanism. No approval, tool call, model request or test was
executed for this source-only clarification.
