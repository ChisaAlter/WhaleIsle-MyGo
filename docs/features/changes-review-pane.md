# Turn changes review pane (DSK18)

Source contract: e9 `packages/client/ui-deliverables/src/client/ReviewTab.tsx`,
`review-definition.ts`, `review-store.ts`, and `index.ts`.

The turn-delivery card now opens `changes-review`, independently of repository
Git diff. Its identity is the recorded Host, viewed session, turn and change
announcement sequence. File selection, split/unified mode and wrapping belong
to each occurrence. Opening another row of the same turn focuses its existing
pane; splitting clones selectors and presentation without sharing mutable state.
Closing removes that occurrence; layout restore saves only source selectors and
presentation and revalidates the current session's recorded declaration before
reading. A changed live binding or history generation requires explicit reopen.

The pane consumes the existing authenticated `changes.summary`, `changes.diff`,
`present.host` and explicit `changes.open` actions through the session store.
It shows one recorded file comparison, counts and all file choices. It supports
split/unified and wrap/no-wrap, full-file preview through the existing Host file
callback, and explicit native open/reveal with busy/error/retry feedback. No
commit, approval, tool run, prompt submission or local filesystem action is added.

Binary/oversized, unavailable/read failures, added/deleted files, coarse whole-file
replacement, unchanged text and the 5000-line display limit are shown. One-sided
comparisons render unified without changing the chosen split preference. The
native patch renderer is reused; this is not a promise of browser Shiki styling.

Focused tests cover validated source binding, independent split controls,
selector-only restore, foreign Host/stale history/revoked declaration rejection,
close/reopen and bounded one-sided patch rendering.
