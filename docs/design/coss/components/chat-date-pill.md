# chat-date-pill

Source: `apps/ui/registry/default/chat/chat-date-pill.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatDatePill`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatDatePill` → `[data-slot=chat-date-pill]` · `<div>` (line 11)

#### base

```txt
pointer-events-none sticky top-4 z-20 mx-auto flex h-6 w-fit items-center justify-center rounded-full border bg-popover/90 px-2 font-medium text-foreground text-xs backdrop-blur-xs
```

**element — default**

Key metrics: display `flex` · height `24px` · width `fit-content` · padding-inline `8px` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `9999px` · background-color `#ffffffe6` / `#1d1d1de6` (popover/90%) · color `#262626` / `#f5f5f5` (foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium) · z-index `20`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `sticky` | `sticky` | `sticky` |
| top | `16px` | `calc(var(--spacing) * 4)` | `top-4` |
| z-index | `20` | `20` | `z-20` |
| margin-inline | `auto` | `auto` | `mx-auto` |
| display | `flex` | `flex` | `flex` |
| height | `24px` | `calc(var(--spacing) * 6)` | `h-6` |
| width | `fit-content` | `fit-content` | `w-fit` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#ffffffe6` / `#1d1d1de6` | `popover/90%` | `bg-popover/90` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| -webkit-backdrop-filter | `blur(4px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-xs` |
| backdrop-filter | `blur(4px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-xs` |

