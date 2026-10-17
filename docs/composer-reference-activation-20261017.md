# Composer reference activation

Original reference: `vendor/deepseek-harness/packages/client/ui-conversation/src/client/input/editor/reference-activation.ts` routes single primary activation through the live source owner without editing the draft. `ui-reference/src/client/index.ts:168-173` opens file references; `ui-skill/src/client/index.ts:202-219` resolves skill paths from its catalog. Session and folder references do not have that file-preview action.

MyGo now routes the reference label's single primary activation through `composer.Model.OnOpenReference`, connected by the real frame to the existing session-owned `OnOpenDeliverable` path. That path retains current-session/Host checks and existing file preview handling. Quoted file references preserve spaces. Skill catalog projections retain their optional source path; only a matching skill with a real path opens. Unknown skills, session/folder references, malformed/stale references, noncollapsed selections and active IME composition do not open or change the draft.

A separate × button removes a reference; the full label never deletes it. Native removal records undo. Existing right-click editing and drag/reorder remain separate gestures. No new RPC or file-opening protocol is added.

Validation is a focused synthetic check only. No desktop runtime, external file opening, original application execution, merge or deployment is performed by this change.
