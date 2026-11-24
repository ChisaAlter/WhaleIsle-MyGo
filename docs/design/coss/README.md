# coss ui design spec (for a native reimplementation)

A machine-readable spec extracted from the **coss ui** design system (Cal.com's system built on Base UI + Tailwind CSS v4). It exists so that a native Go UI toolkit can reproduce coss components pixel-for-pixel without a browser. This directory holds specs only, no runtime code.

## Provenance

| | |
|---|---|
| Upstream repo | https://github.com/cosscom/ui (site: https://coss.com/ui) |
| Pinned commit | `b2336c45f2b7355c42678ce786b6f14d62b819be` (main, 2026-11-17 16:30:42Z, "fix(chat): stabilize iOS Safari focus, replies, and drawer background (#925)") |
| Retrieved | 2026-11-23, as the commit tarball `https://codeload.github.com/cosscom/ui/tar.gz/<sha>`, which has the same content as `raw.githubusercontent.com/cosscom/ui/<sha>/…` |
| Toolchain versions (from upstream `bun.lock`) | `tailwindcss@4.1.17`, `@base-ui/react@1.9.0` |
| License | MIT, © 2025 coss.com and Origin UI. Verbatim copy in [`LICENSE.coss`](LICENSE.coss) |

### Source files used

- `packages/ui/src/styles/globals.css`: every token (`@theme inline`, `:root`, `.dark`, `@layer base`) and the toast keyframes.
- `apps/ui/registry/default/ui/*.tsx`: all **56** components. These are the components shipped through the shadcn registry. Apart from import lines (and import order in `composer.tsx`) they are identical to `packages/ui/src/components/*.tsx`, which I checked with a diff that ignores import lines.
- `apps/ui/registry/default/chat/*.tsx`: all **20** chat parts. `chat/hooks/*.ts` and `chat/lib/*.ts` hold behavior only and have no styles. The `chat-demo/*` directory is demo composition and is not part of the system.
- `apps/ui/content/docs/components/*.mdx` and `apps/ui/content/docs/(root)/{chat,styling,get-started}.mdx`: the MDX sources that render `https://coss.com/ui/docs/components/<name>`. Descriptions, Base UI API links and keyboard mentions come from these. No browser was used.
- `packages/ui/src/fonts/index.ts`, `README.md`, `font-guide.md`, and `apps/ui/app/layout.tsx` (body classes): font setup.
- `tailwindcss@4.1.17`: `theme.css` (default palette and scales) and `preflight.css`. The class → CSS step runs Tailwind's own design-system API (`__unstable__loadDesignSystem(globals.css).candidatesToCss`). Every value therefore comes from the CSS that Tailwind itself generates for the class. It is never typed in by hand.

Raw upstream sources are **not** committed. The only upstream file copied here is the license.

## Files

| File | Content |
|---|---|
| `tokens.json` | All tokens for light and dark. `palette` covers the 244 Tailwind colors plus black and white. `semantic` covers the 41 semantic tokens for both modes. It also holds radius, spacing, breakpoints, containers, typography (fonts, scale, the size/line-height/weight pairs components actually use), shadows (the scale plus every resolved shadow used), focus ring, borders, opacity, z-index, and motion (defaults, easings, animations, keyframes, every transition used, every enter/exit delta). |
| `components/<name>.md` | One file per source file: 56 ui components and 20 chat parts, **76 files** in total. Each file covers anatomy (every exported component and its `data-slot` parts), the cva variant definitions, the exact class strings, resolved values per target and state, mobile differences, and behavior/keyboard notes. |
| `index.json` | The same data in machine-readable form: `components.<name>.parts[].configs[].targets.<target>.{default,states,mobile}`. `usage` holds cross-component aggregates. 567 parts, 686 variant/conditional configs, about 9.7k resolved property values. |
| `generator/` | The extraction scripts, so the output can be re-run against a newer upstream commit (see below). |

## How to read the spec

**Part.** A part is one JSX element that has a `className` or a `data-slot`, for example `MenuPopup → [data-slot=menu-popup]`. Base UI wrappers such as `MenuPrimitive.Positioner` and `MenuPrimitive.Popup` are listed as separate parts in render order. Prop defaults that affect layout (for example `sideOffset=4`, `align="center"`) are listed under **Prop defaults**.

**Config.** `base` holds the part's unconditional classes plus its cva defaults. Each other config adds one cva variant value (`variant=outline`, `size=sm`), one compound variant, or one conditional fragment (`when \`size === "sm"\``). In the markdown, a non-base config shows only the properties that **differ from base**. `index.json` always stores full values.

**Target.** `element` is the part itself. Other targets are pseudo-elements and descendants addressed through Tailwind variants: `::before`, `::after`, `::placeholder`, `svg:not([class*='size-'])` (icons without an explicit size class), `> *[data-slot="…"]` (direct children with that slot), and so on. coss uses `::before` heavily. It is an inset overlay with radius `outer − 1px` that carries the 1px top highlight shadow in light mode and a −1px shadow in dark mode.

**States.** Each state is computed by setting the matching DOM flags and running the cascade again. A state lists only the properties that change compared with the default. Named states group equivalent selectors:

| state | selectors that count |
|---|---|
| hover | `:hover`, honoured only when `(hover: hover)` |
| pressed | `:active`, `[data-pressed]`, plus `:hover` |
| focus-visible | `:focus-visible`, `:focus`, `:focus-within`, `:has(:focus-visible)` (input wrappers) |
| disabled | `:disabled`, `[data-disabled]`, `[aria-disabled=true]`, `:has(:disabled)` |
| invalid | `[aria-invalid]`, `[data-invalid]`, `:has([aria-invalid])` |
| open | `[data-open]`, `[data-popup-open]`, `[data-panel-open]`, `[data-expanded]`, `[aria-expanded=true]` |
| checked / unchecked / indeterminate | `[data-checked]`, `[data-unchecked]`, `[data-indeterminate]` |
| selected | `[data-selected]`, `[data-active]`, `[aria-current=page]`, `[data-state=active]` |
| highlighted | `[data-highlighted]` (keyboard/pointer highlight in menus and lists) |
| starting-style (enter from) | `[data-starting-style]`: the **values the element animates from** on open |
| ending-style (exit to) | `[data-ending-style]`: the **values it animates to** on close |
| rtl | `:dir(rtl)`, `[dir=rtl]` |

Any other atom becomes its own state, named by the selector itself:

- `in(X)` means an ancestor matches X.
- `parent(X)` means the direct parent matches X.
- `prev(X)` means the previous sibling matches X.
- `prevsib(X)` means some earlier sibling matches X.
- `peer(X)` means a Tailwind peer matches X.
- `tag(button)` means the part is rendered as `<button>` through `render`.
- `userclass(w-)` means the consumer passed a `w-*` class. coss uses `not-[class*='w-']` to apply a default width only when the consumer did not set one.

Combined states such as `focus-visible + invalid` or `checked + rtl` are added automatically whenever some rule requires both at once.

**Light/dark.** Values are written as `light / dark`, and as a single value when both modes match. Dark mode is the `.dark` ancestor class (`@custom-variant dark (&:is(.dark *))`).

**src.** `src` names the class that won the cascade for that property. For `box-shadow` it is the last class that set `box-shadow`. The resolved value already combines every `--tw-*` layer, so ring, inset, offset and drop shadows all come out as one list of layers, with transparent and zero layers removed.

**`token`/`expr`.** `token` is filled when the value is a plain token reference: `primary`, or `accent/50%`, which means `accent` at 50% of its own alpha. Otherwise `expr` carries the CSS that Tailwind generated, before variables are substituted.

**Mobile.** The mobile environment is 375px wide with a coarse pointer and no hover. Mobile entries list only what differs from desktop. In most controls this is: height one step taller, text 16px/24px instead of 14px/20px, icons 18px instead of 16px, and a 44×44px `::after` touch target.

## Conversion rules

- **Units.** 1rem = 16px. Tailwind v4 spacing is `calc(var(--spacing) * n)` with `--spacing: 0.25rem`, so `n × 4px` (`h-8.5` → 34px). `--spacing(n)` inside arbitrary values is rewritten the same way. `calc()` is evaluated when every operand is a length or a number. If any operand is a runtime variable or a percentage, the expression stays symbolic, for example `calc(var(--anchor-width) + 20px)`. Unitless line-heights (`--text-sm--line-height: calc(1.25 / 0.875)`) are multiplied by the computed font-size and reported in px.
- **Environment.** Values are resolved for a **desktop** environment: viewport 1280px (`sm`, `md`, `lg` and `xl` media rules apply, `2xl` and above do not), `pointer: fine`, `hover: hover`, motion allowed. `@supports` checks are treated as true, except `-webkit-touch-callout`, which is iOS-only and treated as false. `@container` and `forced-colors` rules are left unevaluated and listed wherever they occur (only `sidebar`).
- **oklch().** Converted with the standard OKLab ⇄ linear-sRGB matrices (Björn Ottosson) and the sRGB transfer curve.
- **Gamut.** 82 palette steps fall outside the sRGB gamut, because Tailwind v4 colors are defined in a wide gamut. `hex` clips each channel, which matches Chromium on an sRGB display and also matches the hex values Tailwind documents (for example red-500 `#fb2c36`, blue-500 `#2b7fff`). `hexGamutMapped` applies the CSS Color 4 OKLCh chroma-reduction algorithm. The two differ by at most 1/255, and only on 4 steps (amber-400/500, yellow-400/500).
- **`--alpha(X / N%)`** (globals) and **`--theme(--color-x/N%)`** (classes) are both `color-mix(in oklab, X N%, transparent)`, which is X with its alpha multiplied by N. Tailwind's opacity modifiers (`bg-primary/90`) work the same way. Alpha is **kept** in the output (`#rrggbbaa`). Do not pre-blend it: composite over whatever is underneath. Many coss surfaces are translucent (`accent`, `muted`, `secondary`, `border`, `input`).
- **`color-mix(in srgb, A p%, B)`** (globals: `muted-foreground`, dark `background`/`card`/`popover`/`sidebar`/`destructive`, `sidebar-foreground`) is premultiplied linear interpolation of gamma-encoded sRGB channels, as the CSS spec defines it. Example: dark `--background` = neutral-950 (#0a0a0a) mixed 96% with white = `#141414`.
- **Relative color** `oklab(from rgb(0 0 0 / 0.05) l a b / 5%)` (Tailwind's `shadow-*/N`) is the source color with alpha N.
- **Cascade.** Rules are ordered by specificity first (`:where()` counts 0; `:is()`, `:not()` and `:has()` count as their most specific argument) and then by Tailwind's own class order. `!important` comes last. Preflight `* { border: 0 solid }` and globals `* { border-color: var(--border); outline-color: ring/50% }` sit underneath every part. They are omitted from tables when the part has no border.
- **`--tw-*` variables** are registered by Tailwind with `inherits: false`, so they are resolved only from the same element or from their `@property` initial value. That is also why `@theme inline { --tw-shadow-color: #000 }` in globals has no effect on components.
- **Custom properties set by an ancestor part** (for example `--thumb-size` set on `switch` and used by `switch-thumb`, or `--cell-size` in calendar) are resolved from the part in the same file that declares them, and the value is tagged `inherited … (inferred: ancestor part)` in `index.json`.

## Values that are runtime inputs, not tokens

These CSS variables are set at runtime by Base UI or by inline style. The spec leaves them symbolic, and a native implementation has to provide them:

- `--anchor-width`, `--anchor-height`, `--available-width`, `--available-height` and `--transform-origin`: from the popup positioner. Popups scale from `--transform-origin`.
- `--accordion-panel-height`, `--collapsible-panel-height`: height animations between 0 and the measured content height.
- `--active-tab-left/top/width/height/bottom` (tabs indicator), `--active-item-*` (segmented control).
- `--nested-dialogs`, `--nested-drawers`: counts of open nested popups. The parent popup scales and dims based on them.
- `--drawer-swipe-progress`, `--drawer-swipe-strength`, `--drawer-swipe-movement-x/y`, `--drawer-height`: drawer gesture.
- `--toast-index`, `--toast-height`, `--toast-offset-y`, `--toast-calc-offset-y`, `--toast-swipe-movement-x/y`, `--toast-frontmost-height`: the toast stack.
- `--scroll-area-overflow-*`: scroll fade masks.
- `--sidebar-width`, `--sidebar-width-icon`: inline style on the sidebar wrapper. The values are module constants listed at the top of `components/sidebar.md`.
- `--popup-width`, `--positioner-width/height`, `--skeleton-width`, `--avatar-group-cut-*`: per-instance inline styles.

## Reproducing

From `generator/` (with Node 20 or newer), with the upstream tarball extracted to `../ui-<sha>` relative to the working directory:

```sh
npm i                         # tailwindcss@4.1.17, typescript@5 (shared npm cache)
node extract.mjs              # TSX AST → extracted.json (parts, cva, conditions)
node build.mjs <outdir>       # components/*.md + index.json (+ usage.json, keyframes.json in cwd)
node tokens.mjs <outdir>      # tokens.json (needs usage.json from build)
```

`keyboard.mjs` holds the hand-written behavior and keyboard notes. Everything else is derived.

## Ambiguities and open questions

1. **`--destructive-foreground` (light).** `globals.css` sets `red-700` (`#c10007`). The manual-install CSS snippet in `alert.mdx` sets `red-400` for `:root`, while the button, badge, menu, context-menu and field snippets all say `red-700`. I treated `globals.css` as authoritative.
2. **Keyboard behavior** is mostly not documented on coss pages. The notes in each component's "Behavior & keyboard" section come from the Base UI primitive or WAI-ARIA pattern being wrapped, and are tagged *inferred*. Exact Base UI defaults (tooltip and preview-card delays, tabs `activateOnFocus`, autocomplete double-Escape) should be checked against Base UI 1.9 before they are reimplemented.
3. **Fonts.** The components use one Cal Sans **variable** file (`CalSansVF.woff2`, weights 400–700) at every size. `font-guide.md` recommends "Cal Sans Text UI" below 20px, but the shipped package does not include that face. Cal Sans is OFL. The license of Paper Mono (the mono font) is not stated in the repo, so it needs checking before redistribution. With no font loaded, the fallback is Tailwind's `ui-sans-serif, system-ui, …` stack.
4. **Dynamic class names.** 12 `className` inputs cannot be resolved statically, for example a class string chosen from a runtime value or the result of another component's helper. Each is listed under "Dynamic className inputs" in its component file. Class strings stored in lookup tables, such as toast positions and sheet sides, are listed under "Other class strings in this file" and resolved there.
5. **Viewport-dependent rules.** `md:`, `lg:` and `xl:` rules (sidebar, sheet, drawer) are resolved at 1280px. A window narrower than 1024px or 768px should apply the raw class strings.
6. **`not-[class*='w-']`.** Tailwind 4.1.17 generates `:not(*:is(class*='w-'))` for this, which looks malformed (the attribute brackets are missing). `:is()` takes a forgiving selector list, so the bad argument is dropped and the selector matches every element. In practice the default (for example the menu popup's `min-w-32` = 128px) therefore **always** applies, even when the consumer passes `w-*`. The spec records it as the default state, plus a `userclass(w-)` state that removes it, which is the evident intent. Treat that state as intent-only, not as shipped behavior.
7. **Autofill.** `[transition:background-color_5000000s_ease-in-out_0s]` on inputs is a hack that stops the browser's autofill background from showing. It is not a real transition.
8. **Selector modelling.** A few deeply nested arbitrary variants (segmented-control indicator sibling logic, group separators that react to hovered neighbours, `has(peer(+…))` in select) are modelled as opaque atoms. The raw class strings are always printed next to them, so in those cases the class string is the source of truth.
9. **Container and forced-colors queries** (sidebar only) are not evaluated.
