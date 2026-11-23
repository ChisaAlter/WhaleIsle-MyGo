# Frame

Source: `apps/ui/registry/default/ui/frame.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/frame.mdx`

> A framed container for grouping related information.

Depends on: —

Exports: `Frame`, `FramePanel`, `FrameHeader`, `FrameTitle`, `FrameDescription`, `FrameFooter`

## Behavior & keyboard

- Presentational framed container (panels separated by borders); no keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Frame` → `[data-slot=frame]` · `<div>` (line 9)

#### base

```txt
relative flex flex-col rounded-2xl bg-muted/72 p-1 *:[[data-slot=frame-panel]+[data-slot=frame-panel]]:mt-1
```

**element — default**

Key metrics: display `flex` · padding `4px` · border-radius `16px` (radius-2xl) · background-color `#00000007` / `#ffffff07` (muted/72%)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `bg-muted/72` |
| padding | `4px` | `calc(var(--spacing) * 1)` | `p-1` |

**`> *:is([data-slot=frame-panel]+[data-slot=frame-panel])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-top | `4px` | `calc(var(--spacing) * 1)` | `*:[[data-slot=frame-panel]+[data-slot=frame-panel]]:mt-1` |

### `FramePanel` → `[data-slot=frame-panel]` · `<div>` (line 26)

#### base

```txt
relative rounded-xl border bg-background bg-clip-padding p-5 shadow-xs/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-xl)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: padding `20px` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `14px` · background-color `#ffffff` / `#141414` (background) · box-shadow `0 1px 2px 0 #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| border-radius | `14px` | `calc(var(--radius) + 4px)` | `rounded-xl` |
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |
| background-clip | `padding-box` | `padding-box` | `bg-clip-padding` |
| padding | `20px` | `calc(var(--spacing) * 5)` | `p-5` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `before:rounded-[calc(var(--radius-xl)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `FrameHeader` → `[data-slot=frame-panel-header]` · `<header>` (line 42)

#### base

```txt
flex flex-col px-5 py-4
```

**element — default**

Key metrics: display `flex` · padding-inline `20px` · padding-block `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| padding-inline | `20px` | `calc(var(--spacing) * 5)` | `px-5` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

### `FrameTitle` → `[data-slot=frame-panel-title]` · `<div>` (line 55)

#### base

```txt
font-semibold text-sm
```

**element — default**

Key metrics: font-size `14px` (text-sm) · line-height `20px` · font-weight `600` (font-weight-semibold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| font-weight | `600` | `font-weight-semibold` | `font-semibold` |

### `FrameDescription` → `[data-slot=frame-panel-description]` · `<div>` (line 68)

#### base

```txt
text-muted-foreground text-sm
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `FrameFooter` → `[data-slot=frame-panel-footer]` · `<footer>` (line 81)

#### base

```txt
px-5 py-4
```

**element — default**

Key metrics: padding-inline `20px` · padding-block `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `20px` | `calc(var(--spacing) * 5)` | `px-5` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

