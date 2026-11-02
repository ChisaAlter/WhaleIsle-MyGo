# Chat chrome residual closure — S27 / S28 (2026-10-26)

Scope: the two code-fixable residuals recorded in
`docs/parity/gap-matrix-20261025.json`:

- **S28** — composer card 772 vs body 774.4 DIP (2.4 DIP) and the navigation label
  baseline 1.7 DIP low. Round 156 closed the card height; rounds 177/210 fixed the
  nav line box and the axis root cause, but no test asserted a **0** difference.
- **S27** — the desktop hero had no defined vertical seat; it was the remainder
  of centring a stack whose card height had changed.

Everything below is a headless layout measurement from `ui.NewTester` (the
package's existing tester pattern), not a screenshot. Native numbers are the UIA
probes quoted in `docs/status.md` round152/round181/round210/round232/round245.

## 1. Where the numbers come from

| seat | code owner | native evidence |
| --- | --- | --- |
| Window / column sizes | `cmd/whaleisle/main.go:2537` (`state.layout = frame.DefaultLayout()`); `internal/ui/frame` columns | 1440x920 main window; sidebar 280; conversation column 1160 |
| Content axis | `internal/ui/frame/frame.go` (`conversationContentWidth`) | `0.64 × 1160 = 742.4` (round186/210) |
| Composer body | `internal/ui/frame/chat_chrome_seats.go` (`conversationComposerBodyWidth`) | `742.4 + 2×16 = 774.4`, the installed `max-width: 774.4px` |
| Composer card | `internal/ui/composer/composer.go` (16-DIP side clearance inside the stack) | card 774.39 × 155.34 at (469.3, 409.33) |
| Nav label line box | `internal/ui/sidebar/sidebar.go` (`FixedLineHeight(22)` under 7-DIP row padding) | 插件 label y=127, h=22; ink centre 138 (was 1.7 low with an 18.7 box) |
| Hero vertical seat | `internal/ui/frame/conversation_phase.go` (`conversationHero*` constants) + `frame.go` top block/foot | slogan 314.67, preset 373.33, card 409.33 at 1440×860; stack ends at 564.67 |

`cmd/whaleisle/main.go` owns only the window and the `DefaultLayout()` seed; the
seats themselves live in `internal/ui/frame`, so no window option changed.

## 2. Before → after (DIP)

| metric | before | after | native | delta |
| --- | ---: | ---: | ---: | ---: |
| Composer card width | 772.00 (round156 report) | **774.400** | 774.39 / 774.4 | **0.000** vs the body seat (`conversationComposerBodyWidth(742.4) = 774.400024`) |
| Composer card x | 470.7 (round152) | 469.80 | 469.3 | +0.50 (divider-rounded centre; not part of S28) |
| Nav label box height | 18.7 | **22.00** | 22 | **0.00** |
| Nav label y | not asserted (18.7 box) | **127.00** | 127 | **0.00** |
| Nav label centre / baseline | 1.7 DIP low | **138.00** | 138 | **0.00** |
| Nav row pitch | 36 → 40 (round152) | 40.00 | 40 | 0.00 |
| Hero slogan y @1440×860 | 314.85 | **314.67** | 314.67 | 0.00 |
| Hero slogan y @1440×920 | 344.85 | **344.67** | 344.67 (+30) | 0.00 |
| Hero preset y @1440×860 | 373.15 | **373.33** | 373.33 | 0.00 |
| Hero preset y @1440×920 | 403.15 | **403.33** | 403.33 (+30) | 0.00 |
| Hero card top @1440×860 | 409.15 | **409.33** | 409.33 | 0.00 |
| Hero card top @1440×920 | 439.15 | **439.33** | 439.33 (+30) | 0.00 |
| Hero stack height | not defined | **250.00** | 250 (314.67 → 564.67) | 0.00 |

The 0.18-DIP hero error was not a centring bug: the headline foot was 22.3
(22.32 rounded to 0.1), while the native stack needs 22.66 =
250 − 32 headline − 40 preset row (4+28+8) − 155.34 card. With 22.66 the
three seats land exactly and the stack is a fixed 250-DIP block.

## 3. Code changes

- New `internal/ui/frame/chat_chrome_seats.go`
  - `conversationComposerClearance = 16`
  - `conversationNavRowLineBox = 22`, `conversationNavLabelCentre = 138`
  - `conversationComposerBodyWidth(axis)` / `conversationComposerStackWidth(axis)`
- `internal/ui/frame/frame.go`
  - the composer stack is now `conversationComposerStackWidth(contentAxis)`
    (was the bare `contentAxis+64`); the card therefore fills exactly
    `conversationComposerBodyWidth(contentAxis)`.
  - the hero top block uses the named `conversationHeroTopBlock` instead of a
    literal `41.34`.
- `internal/ui/frame/conversation_phase.go`
  - `conversationHeroHeadlineFoot` 22.3 → **22.66**
  - named `conversationHeroTopBlock` (41.34), `conversationHeroHeadlineHeight`
    (32), `conversationHeroPresetRowHeight` (40), `conversationHeroCardHeight`
    (155.34), `conversationHeroStackHeight` (their sum = 250).
- New tests in `internal/ui/frame/chat_chrome_residual_test.go`
  - `TestComposerCardMatchesTheConversationBodyWidth` — card vs body = 0.000
    DIP (asserted < 0.001); body vs native 774.4 < 0.01; card x 469.3 ± 0.6.
  - `TestNavigationLabelKeepsTheNativeBaseline` — label h=22, y=127,
    centre−138 = **0**, row pitch = 40.
  - `TestHeroStackHasADefinedNativeSeat` — at 1440×860 and 1440×920 the
    slogan/preset/card are 314.67/373.33/409.33 and 344.67/403.33/439.33
    (±0.02), card h=155.34, stack=250.00, preset→card gap=8.
- Updated `internal/ui/frame/hero_headline_height_test.go` to pin 22.66 with the
  250-DIP derivation instead of the old 22.32/22.3 rounding.

## 4. Verification

```
go test ./internal/ui/frame -run 'TestComposerCardMatchesTheConversationBodyWidth|TestNavigationLabelKeepsTheNativeBaseline|TestHeroStackHasADefinedNativeSeat' -count=1 -v
--- PASS: TestComposerCardMatchesTheConversationBodyWidth
--- PASS: TestNavigationLabelKeepsTheNativeBaseline
--- PASS: TestHeroStackHasADefinedNativeSeat
--- PASS: TestHeroStackHasADefinedNativeSeat/1440x860
--- PASS: TestHeroStackHasADefinedNativeSeat/1440x920
```

The whole `./internal/ui/frame` suite is green as well, including the existing
hero/docked width-identity tests (`TestHeroComposerCardUsesTheNativeColumnAxis`,
`TestHeroStackKeepsTheNativeSeat`, `TestSidebarChromeMatchesTheNativePage`).

## 5. What is still outside this proof

- These are layout (DIP) assertions, not a fresh 150%-DPI screenshot pair; the
  gap matrix's pixel caveat for D-MAIN still applies to font rasterisation and
  the wallpaper layer.
- The 0.5-DIP composer x offset (469.80 vs 469.3) comes from the divider-rounded
  `Sizes.Center`; it is not the S28 width residual and was left unchanged.
- `composerCardMaxWidth` (774.39) remains the measured-card documentation
  constant; the live card is bounded by the body/seat chain, so a very wide
  window still follows the transcript axis rather than the native literal
  `max-width` (the known round186 open question).
