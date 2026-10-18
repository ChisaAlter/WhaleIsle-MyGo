# Account contact and frame-wide notices

Restores the account feature's desktop entry flows from the original ui-settings-account package:

- AccountMenu contact action → sidebar footer and Account settings contact action. Both signed-in and signed-out users open the actual configured-default Feishu questionnaire through the existing desktop external opener. Only an explicit click samples display-safe UID, build, OS/architecture and display resolution; no credential is accepted by the URL helper. The form's original hidden/prefill field names and unsupported-field cleanup are retained.
- AccountMenu session-expired/model-sign-in-required subscriptions → existing Host event stream, usage store revision and application-root native toasts. No automatic login, retry or polling is introduced.
- AccountQuotaNotice registration for shell.quota-notice code ACCOUNT_QUOTA → application-root native modal. Quota is sourced only from newly accepted live turn/end error events, after durable-event deduplication. Historical snapshots/pages never raise it. The notice survives changing panels/sessions; explicit edit of the affected user turn retires it.
- A signed-in account with a top-up destination gets Cancel / Top up. Top up holds the current notice across later quota events, invokes the existing external recharge flow only after clicking, retains retry/error feedback on opening failure, and dismisses on return before the existing account refresh. Sign-out/Host teardown retire claims. QUOTA, signed-out or unavailable top-up states use the generic warning.

Original references (under vendor/deepseek-harness/packages/client):

- ui-settings-account/src/client/AccountMenu.tsx
- ui-settings-account/src/client/AccountQuotaNotice.tsx
- ui-settings-account/src/client/index.ts (contactUs and slot registrations)
- ui-settings-account/src/contact-config.ts and src/client/contact-url.ts
- ui-chat/src/client/apply.ts (live append quota owner and holds)

Offline focused checks supplied, not run by the implementing worker:

- internal/accountcontact: TestContactClickUsesSampledContextAndOptionalDevice
- internal/store/session: TestQuotaNoticeOnlyLiveAppendAndHeldUntilReturn

No live account, feedback submission, login, recharge, browser launch, GUI, build or service was exercised. This uses the application's existing external recharge return flow rather than replacing its webview/account security policy.
