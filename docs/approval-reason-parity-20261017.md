# CH54: approval reason follows current UI language

## Source contract
Original `vendor/deepseek-harness/packages/client/ui-approval/src/client/ApprovalPanel.tsx:17-19` chooses displayReason when supplied, using current locale via `ui-approval/src/client/index.ts:99-101`. `locale/src/client/index.ts:208-220` preserves plain strings, chooses the active-language/fallback value, and preserves explicitly empty localized strings.

## Native implementation
- `internal/store/interaction/store.go:28-31,557-569`: retain raw displayReason JSON and original reason as immutable strings on the real pending waterfall. Existing Detail compatibility remains unchanged.
- `internal/ui/interaction/approval_reason.go:12-38`: resolve at render time using current i18n locale, regional/base language and English fallback. Plain strings remain verbatim; explicitly empty translations remain empty; malformed/missing data falls back to original reason or legacy Detail.
- `approval_reason.go:41-52`: keep original reason separately visible when the localized display differs; its label follows current UI language.
- `internal/ui/interaction/card.go:136-141`: approval presentation uses the resolver without changing allow/deny/args/session ownership.

No translation service, network request, new language catalog, or automatic approval is introduced. All locale alternatives and original reason survive switching UI language while the approval remains open.

## Verification
`TestApprovalReasonFollowsLocaleWithoutLosingOriginal` first failed on the real waterfall/UI path: Chinese locale was ignored and original reason lost. Final focused store/UI/Markdown check passed (0.115s / 0.185s / 0.046s). New tests cover Chinese/English switching, original payload retention, regional/base/English fallback, empty translations, plain verbatim strings, malformed/null payload, legacy Detail, and unchanged explicit denial.

No original execution, full-project suite, real Host integration, or target-platform GUI acceptance. Published approval state still requires runtime acceptance.
