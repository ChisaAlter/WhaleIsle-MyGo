// Package lang is a native Go runtime for OpenUI Lang, the generative UI
// language of thesysdev/openui (MIT): parser, streaming parser, evaluator,
// reactive state, actions and persistence. It renders nothing; it produces a
// renderer-neutral element tree for a native renderer.
//
// # Version
//
// The Whale UI viewer bundled in internal/desktop/openui/viewer/v2 derives
// its language runtime from @openuidev/lang-core 1.0.0 (tag commit
// f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20). This package ports that
// version and passes the OpenUI 1.0 conformance fixtures vendored in
// testdata/spec (thesysdev/openui@06ecd41c). Whale UI's deliberate runtime
// changes are selected with Profile (WhaleProfile); the zero Profile is the
// upstream behavior.
//
// # Pipeline
//
//	text ──Preprocess──▶ statements ──parse──▶ AST ──materialize──▶ ParseResult
//	                                     (StreamParser reuses completed statements)
//	ParseResult + Store ──Instance.Render──▶ Frame{Root *Element, Errors}
//	*ActionPlan ──Instance.Dispatch──▶ ActionRun.Next() ──▶ host Effects
//
// Untrusted input is bounded by Limits; every overrun becomes a
// budget-exceeded Diagnostic, never a panic. Recovery follows spec section 8:
// the smallest unit is dropped and reported.
//
// # API summary
//
//   - LoadLibrarySpec builds a *Library from a LibrarySpec document
//     (JSON Schema $defs, functions, actions, prompt signatures with
//     $binding markers). Package whaleui provides the generated Whale UI
//     library.
//   - Parse (batch) and NewStreamParser (incremental, spec section 4).
//   - NewInstance / Instance.Update(Update{Revision, Program, Streaming})
//     returns a Frame. Revisions lower than the last applied one are stale
//     and ignored (protocol Render.Revision never goes backwards).
//   - Element trees hold resolved props; bindable props hold *Binding
//     handles (write with Instance.SetBinding); action props hold
//     *ActionPlan values (run with Instance.Dispatch); form fields write
//     with Instance.SetField. Element.FormName is the enclosing form scope.
//   - ActionRun.Next applies @Set/@Reset locally and returns host Effects
//     one at a time (ToAssistant, OpenUrl with http(s)-without-credentials
//     validation, CopyToClipboard, custom actions, mutation runs). Step
//     arguments are evaluated when the step runs. ActionRun.Fail stops the
//     remaining steps; ResolveMutation delivers a mutation result.
//   - Query statements (when Profile.AllowData) surface as FetchRequests
//     through Instance.PendingFetches and resolve with ResolveFetch.
//   - Instance.PersistState / RestoreState read and write the
//     openui-state-v1 document {"format":"whale-ui-state","version":1,
//     "values":{...}} within MaxStateBytes (48 KiB); the legacy flat format
//     is accepted on restore.
//   - MergeStatements (edit mode, spec section 7), Serialize (section 9.2),
//     ValidateRules (section 5.4), StripFences and Preprocess (section 1.2).
//
// # Deviations
//
// From upstream lang-core 1.0.0, in every profile:
//   - Action step arguments (@ToAssistant, @OpenUrl, custom actions) are
//     evaluated when the step runs, as in the Whale viewer, not at render.
//   - A plain object read from data never becomes a custom action step;
//     only {type:"continue_conversation"|"open_url"} data steps are honored
//     inside Action([...]).
//   - String collation for @Sort uses the CLDR root collation
//     (golang.org/x/text/collate) in place of the host's localeCompare.
//   - Member access reads own properties only (objects, array index and
//     length, string index and length).
//   - Expression state defaults (upstream profile) are evaluated once with
//     the built-ins (spec 1.1 wording) instead of storing an AST.
//   - A lone surrogate in a \u escape becomes U+FFFD (Go strings are UTF-8).
//
// WhaleProfile versus spec 1.0 (these are why the fixtures run with the
// upstream profile):
//   - Unicode letters are accepted in names; other unknown characters are
//     skipped (the viewer throws a syntax error on them).
//   - anyOf/oneOf/allOf and JSON Schema type arrays are validated
//     (StrictUnions), so a bare list of actions in an action slot is a
//     type-mismatch rather than plain data (spec 6.3, fixture
//     actions/011).
//   - $variable defaults must be literals (invalid-state-default).
//   - Completed streams add incomplete-input and unresolved-reference
//     diagnostics; evaluated props are validated again at render time.
//   - Query and Mutation report data-unavailable and never fetch.
package lang
