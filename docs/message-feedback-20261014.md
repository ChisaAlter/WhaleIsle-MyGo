# Authoritative per-message feedback
Source: fixed Electron vendor/deepseek-harness/packages/feedback/message-feedback/src/types.ts and client/ui-message-feedback/{controller,MessageFeedbackActions,FeedbackDialog}. Current native feedback.go used outdated up/down, empty ifVersion and fake local success; it also wrote a second sessionFeedback remark.

Replacement:
- Exact messageFeedback/list/put/delete typed business envelope, positive/negative, null create version and observed opaque edit/delete version.
- Only successful acknowledged server values update rating. Missing plugin/load error shows unavailable with read retry. No offline success, no extra sessionFeedback/record, no prompt/model call.
- Like/dislike opens a message-specific note/category draft. Explicit submit writes; clicking an already active rating retracts with its observed version. Existing rating has Edit feedback.
- Categories are original seven category IDs; blank means omitted. Optional note preserves exact user text and is not synthesized.
- Business failure/transport error retains server rating and editor draft. Version conflict incorporates authoritative current item but retains draft for explicit subsequent action. Cancel changes no server data.
- Store/client/selected-session generation rejects late load/write results. Session selection clears the old session editor/current-state namespace.
- Old fixture requiring fake-local success was replaced with one targeted TestMessageFeedbackAuthoritativeFixture: authentic list/create/edit-conflict/delete, null/opaque CAS, no write on editor open, retained draft, cancel, no unavailable fake success.
- Fixture passed. No actual feedback/Host/session/model used. Windows cumulative compile delegated to integrator.

Delivery: replace feedback.go and feedback_test.go; add feedback_editor.go; apply integration.patch (Store field and chat/frame wiring). Existing legacy helper UI is unused by production message rows.
