# Preview Card

Source: `apps/ui/registry/default/ui/preview-card.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/preview-card.mdx` · Base UI: https://base-ui.com/react/components/preview-card#api-reference

> A popup that appears when a link is hovered, showing a preview for sighted users.

Depends on: `@base-ui/react/preview-card`

Exports: `PreviewCardTrigger`, `PreviewCardPopup`

## Behavior & keyboard

- Opens on hover/focus of the trigger link after a delay (Base UI default ~600ms open, ~300ms close); `Escape` closes.
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `PreviewCardTrigger` → `[data-slot=preview-card-trigger]` · `<PreviewCardPrimitive.Trigger>` (line 14)

### `PreviewCardPopup` → `[data-slot=preview-card-positioner]` · `<PreviewCardPrimitive.Positioner>` (line 34)

Attributes: `align={align}`, `anchor={anchor}`, `sideOffset={sideOffset}`

Prop defaults: `align="center"`, `sideOffset=4`

#### base

```txt
z-50
```

**element — default**

Key metrics: z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `50` | `50` | `z-50` |

### `PreviewCardPopup` → `[data-slot=preview-card-content]` · `<PreviewCardPrimitive.Popup>` (line 41)

Prop defaults: `align="center"`, `sideOffset=4`

#### base

```txt
relative flex w-64 origin-(--transform-origin) text-balance rounded-lg border bg-popover not-dark:bg-clip-padding p-4 text-popover-foreground text-sm shadow-lg/5 transition-[scale,opacity] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] data-ending-style:scale-98 data-starting-style:scale-98 data-ending-style:opacity-0 data-starting-style:opacity-0 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · width `256px` · padding `16px` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · transition-property `scale, opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| width | `256px` | `calc(var(--spacing) * 64)` | `w-64` |
| transform-origin | `var(--transform-origin)` | `runtime var --transform-origin` | `origin-(--transform-origin)` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| padding | `16px` | `calc(var(--spacing) * 4)` | `p-4` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| text-wrap | `balance` | `balance` | `text-balance` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `scale, opacity` | `scale,opacity` | `transition-[scale,opacity]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[scale,opacity]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[scale,opacity]` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | scale | `98% 98%` | `data-ending-style:scale-98` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `data-starting-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

