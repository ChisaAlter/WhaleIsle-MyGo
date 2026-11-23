# Meter

Source: `apps/ui/registry/default/ui/meter.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/meter.mdx` · Base UI: https://base-ui.com/react/components/meter#api-reference

> A graphical display of a numeric value within a range.

Depends on: `@base-ui/react/meter`

Exports: `Meter`, `MeterLabel`, `MeterTrack`, `MeterIndicator`, `MeterValue`

## Behavior & keyboard

- `role=meter`; read-only, not focusable.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Meter` → (no data-slot) · `<MeterPrimitive.Root>` (line 13)

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

### `MeterLabel` → `[data-slot=meter-label]` · `<MeterPrimitive.Label>` (line 33)

#### base

```txt
font-medium text-foreground text-sm
```

**element — default**

Key metrics: color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

### `MeterTrack` → `[data-slot=meter-track]` · `<MeterPrimitive.Track>` (line 46)

#### base

```txt
block h-2 w-full overflow-hidden bg-input
```

**element — default**

Key metrics: display `block` · height `8px` · width `100%` · background-color `#0000001a` / `#ffffff14` (input)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `block` | `block` | `block` |
| height | `8px` | `calc(var(--spacing) * 2)` | `h-2` |
| width | `100%` | `100%` | `w-full` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

### `MeterIndicator` → `[data-slot=meter-indicator]` · `<MeterPrimitive.Indicator>` (line 59)

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

### `MeterValue` → `[data-slot=meter-value]` · `<MeterPrimitive.Value>` (line 72)

#### base

```txt
text-foreground text-sm tabular-nums
```

**element — default**

Key metrics: color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| font-variant-numeric | `tabular-nums` | `var(--tw-ordinal,) var(--tw-slashed-zero,) var(--tw-numeric-figure,) var(--tw-numeric-spacing,) var(--tw-numer` | `tabular-nums` |

