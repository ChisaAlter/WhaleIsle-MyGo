# Progress

Source: `apps/ui/registry/default/ui/progress.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/progress.mdx` · Base UI: https://base-ui.com/react/components/progress#api-reference

> Displays the status of a task that takes a long time.

Depends on: `@base-ui/react/progress`

Exports: `Progress`, `ProgressLabel`, `ProgressTrack`, `ProgressIndicator`, `ProgressValue`

## Behavior & keyboard

- `role=progressbar`; not focusable; indicator width follows value; `data-complete`/`data-progressing`/`data-indeterminate` states. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Progress` → `[data-slot=progress]` · `<ProgressPrimitive.Root>` (line 13)

#### base

```txt
flex w-full flex-col gap-2
```

**element — default**

Key metrics: display `flex` · width `100%` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

### `ProgressLabel` → `[data-slot=progress-label]` · `<ProgressPrimitive.Label>` (line 34)

#### base

```txt
font-medium text-sm
```

**element — default**

Key metrics: font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |

### `ProgressTrack` → `[data-slot=progress-track]` · `<ProgressPrimitive.Track>` (line 47)

#### base

```txt
block h-1.5 w-full overflow-hidden rounded-full bg-input
```

**element — default**

Key metrics: display `block` · height `6px` · width `100%` · border-radius `9999px` · background-color `#0000001a` / `#ffffff14` (input)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `block` | `block` | `block` |
| height | `6px` | `calc(var(--spacing) * 1.5)` | `h-1.5` |
| width | `100%` | `100%` | `w-full` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

### `ProgressIndicator` → `[data-slot=progress-indicator]` · `<ProgressPrimitive.Indicator>` (line 63)

#### base

```txt
bg-primary transition-all duration-500
```

**element — default**

Key metrics: background-color `#262626` / `#f5f5f5` (primary) · transition-property `all` · transition-duration `500ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#262626` / `#f5f5f5` | `primary` | `bg-primary` |
| transition-property | `all` | `all` | `transition-all` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-all` |
| transition-duration | `500ms` | `500ms` | `duration-500` |

### `ProgressValue` → `[data-slot=progress-value]` · `<ProgressPrimitive.Value>` (line 76)

#### base

```txt
text-sm tabular-nums
```

**element — default**

Key metrics: font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| font-variant-numeric | `tabular-nums` | `var(--tw-ordinal,) var(--tw-slashed-zero,) var(--tw-numeric-figure,) var(--tw-numeric-spacing,) var(--tw-numer` | `tabular-nums` |

