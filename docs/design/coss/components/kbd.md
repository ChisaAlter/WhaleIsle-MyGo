# Kbd

Source: `apps/ui/registry/default/ui/kbd.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/kbd.mdx`

> A component for displaying keyboard keys and shortcuts.

Depends on: —

Exports: `Kbd`, `KbdGroup`

## Behavior & keyboard

- Presentational `<kbd>`; no behavior.

Docs mentions:

- description: A component for displaying keyboard keys and shortcuts.
- Displays a single keyboard key.
- Groups multiple keyboard keys for key combinations.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Kbd` → `[data-slot=kbd]` · `<kbd>` (line 9)

#### base

```txt
pointer-events-none inline-flex h-5 min-w-5 select-none items-center justify-center gap-1 rounded-[.25rem] bg-muted px-1 font-medium font-sans text-muted-foreground text-xs [&_svg:not([class*='size-'])]:size-3
```

**element — default**

Key metrics: display `inline-flex` · height `20px` · min-width `20px` · padding-inline `4px` · gap `4px` · border-radius `4px` · background-color `#0000000a` / `#ffffff0a` (muted) · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `20px` | `calc(var(--spacing) * 5)` | `h-5` |
| min-width | `20px` | `calc(var(--spacing) * 5)` | `min-w-5` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| border-radius | `4px` | `.25rem` | `rounded-[.25rem]` |
| background-color | `#0000000a` / `#ffffff0a` | `muted` | `bg-muted` |
| padding-inline | `4px` | `calc(var(--spacing) * 1)` | `px-1` |
| font-family | `ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` | `font-sans` | `font-sans` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `12px` | `calc(var(--spacing) * 3)` | `[&_svg:not([class*='size-'])]:size-3` |
| height | `12px` | `calc(var(--spacing) * 3)` | `[&_svg:not([class*='size-'])]:size-3` |

### `KbdGroup` → `[data-slot=kbd-group]` · `<kbd>` (line 25)

#### base

```txt
inline-flex items-center gap-1
```

**element — default**

Key metrics: display `inline-flex` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

