# Round114 exact installed ToolRow controlled Chrome

PASSED: 9/9 cases, 42/42 assertions. [Measured JSON](tool-row-pass258-reference-2026-10-24T00-04-27-928Z-9d75a2f8.json) · [Summary](tool-row-pass258-reference-2026-10-24T00-04-27-928Z-9d75a2f8.txt).

Exact installed ToolRow, DisclosureRow/TextShimmer CSS plus installed theme base/platform/font-axis literals. Installed wrapper relationships manually assembled; actual icon paths verified in installed web bundle. Fixture-only synchronous click/keyboard handlers. No React, original business toggler, full app, live Session or original usePresence/motion claim.

[Idle](tool-row-pass258-reference-2026-10-24T00-04-27-928Z-9d75a2f8-idle.png) · [Trusted hover](tool-row-pass258-reference-2026-10-24T00-04-27-928Z-9d75a2f8-hover.png) · [Open generic IN/OUT](tool-row-pass258-reference-2026-10-24T00-04-27-928Z-9d75a2f8-open.png) · [Body font18](tool-row-pass258-reference-2026-10-24T00-04-27-928Z-9d75a2f8-font18.png)

At process DPR1: row24, leading16, glyph14, title/summary13px/24px, leading→title gap6. Idle row/title/summary rgb(129, 133, 140); trusted hover rgb(97, 102, 107); open without hover rgb(129, 133, 140). Background remains transparent. Idle business icon changes to down chevron on hover; open uses up chevron.

Generic card used border1px, radius16px, margin4px 0px 4px 4px, dimensions596×82.5. Section outer heights40/40, each grid padding12 16/gap14, max-height150 with observed content-box. Divider height0.5 (not a border). Card font "SF Mono", "JetBrains Mono", "Fira Code", Consolas, "Liberation Mono", Menlo, Courier, "PingFang SC", "Microsoft YaHei", "SF Mono", "JetBrains Mono", "Fira Code", Consolas, "Liberation Mono", Menlo, Courier, "PingFang SC", "Microsoft YaHei" 11px/16px. No generic copy buttons; optional Inspect is omitted because no callback was provided. English labels IN/OUT; Chinese 输入/输出.

Preparing retains its business icon and has no role/tabindex/aria-expanded/data-expandable; handlers are attached only to expandable rows. Error summary/OUT use error-primary `#ec1313`; stopped summary uses warn-label `#dd8629`. Fixture strings are synthetic and localization labels are read from installed dictionaries. Long input and long output are measured separately; section scrollHeight/clientHeight and trusted input-wheel result are retained. Body font18 applies the installed font axis (delta4), measuring row28/leading20/glyph18 and secondary16. The code-small tier remains11/16 because this changes the body font setting, not the independent code font setting.

| Measurement at process DPR1 | Observed |
| --- | --- |
| Single-line input / output section outer height |40 /40px|
| Long input / long output section outer height |174 /174px|
| Long section max content height |150px plus12px top and12px bottom padding|
| Generic short card outer height |1 +40 +0.5 +40 +1 =82.5px|
| Expanded own card total height |24px header +4px top margin +82.5px card +4px bottom margin =114.5px|
| IN / OUT gutter-label widths |12.109375 /18.15625px|
| Caption / payload color |`#adb2b8` /`#61666b`|
| Card background / border |`#f9fafb` /rgba(0,0,0,.04)|

Each section owns its own max-content label column, so IN and OUT payloads need not start at the same X. Label origin is section-left16px, and payload begins14px after that section's measured label width. The0.5px divider occupies layout height0.5px; it is not snapped like the declared0.5px border, which reserves1CSSpx at this process scale. The JSON is222,523bytes and records measurements once under named states rather than repeating complete DOM/style snapshots for every assertion.

Source extraction includes exact CSS, installed primitive slice offsets/hash, token literal hashes, installed ToolRow source line references, and reference SVG path data individually verified in the installed web bundle. Before/after hashes cover every read installed source plus both runners/reference icon files. Browser startup flag/PID/profile/port are verified via CDP before any page input; cleanup records own processes gone, port released and unique profile removed. No original GUI, profile, foreground, lock, runtime or model operation occurred.

- Synthetic tool names/payloads and local handler state are fixture-only; exact CSS/DOM/icon geometry is the measured subject.
- Fonts resolve installed token families on this system; no live persisted user theme/font preferences or bundled remote fonts are loaded.
- Trusted CDP events exercise fixture handlers and browser CSS, not original React dispatch, useDisclosure or usePresence.
- Process scale1 is Chrome startup configuration; no Emulation calls, physical-monitor, hardware input or Electron/full-app claim.

Every browser case ran once. Failures remain in raw JSON/text. No rerun is made to obtain a passing result.
