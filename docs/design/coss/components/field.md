# Field

Source: `apps/ui/registry/default/ui/field.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/field.mdx` · Base UI: https://base-ui.com/react/components/field#api-reference

> A component that provides labelling and validation for form controls.

Depends on: `@base-ui/react/field`

Exports: `Field`, `FieldLabel`, `FieldItem`, `FieldDescription`, `FieldError`

## Behavior & keyboard

- Wires label, control, description and error (`aria-describedby`, `aria-invalid`); validation state shown via `data-invalid`/`data-valid`/`data-touched`/`data-dirty`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Field` → `[data-slot=field]` · `<FieldPrimitive.Root>` (line 12)

#### base

```txt
flex flex-col items-start gap-2
```

**element — default**

Key metrics: display `flex` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

### `FieldLabel` → `[data-slot=field-label]` · `<FieldPrimitive.Label>` (line 25)

#### base

```txt
inline-flex items-center gap-2 font-medium text-base/4.5 text-foreground data-disabled:opacity-64 sm:text-sm/4
```

**element — default**

Key metrics: display `inline-flex` · gap `8px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| font-size | `14px` | `text-sm` | `sm:text-sm/4` |
| line-height | `16px` | `calc(var(--spacing) * 4)` | `sm:text-sm/4` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: font-size `16px` · line-height `18px`

### `FieldItem` → `[data-slot=field-item]` · `<FieldPrimitive.Item>` (line 41)

#### base

```txt
flex
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |

### `FieldDescription` → `[data-slot=field-description]` · `<FieldPrimitive.Description>` (line 54)

#### base

```txt
text-muted-foreground text-xs
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `FieldError` → `[data-slot=field-error]` · `<FieldPrimitive.Error>` (line 67)

#### base

```txt
text-destructive-foreground text-xs
```

**element — default**

Key metrics: color `#c10007` / `#ff6467` (destructive-foreground) · font-size `12px` (text-xs) · line-height `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| color | `#c10007` / `#ff6467` | `destructive-foreground` | `text-destructive-foreground` |

