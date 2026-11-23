# Alert

Source: `apps/ui/registry/default/ui/alert.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/alert.mdx`

> A callout for displaying important information.

Depends on: —

Exports: `Alert`, `AlertTitle`, `AlertDescription`, `AlertAction`

## Behavior & keyboard

- Static `role=alert` container; no keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `alertVariants` (line 5)

Defaults: {"variant":"default"}

```txt
base: relative grid w-full items-start gap-x-2 gap-y-0.5 rounded-xl border px-3.5 py-3 text-card-foreground text-sm has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto] has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr] has-data-[slot=alert-action]:grid-cols-[1fr_auto] has-[>svg]:gap-x-2 [&>svg]:h-lh [&>svg]:w-4
variant.default: bg-transparent dark:bg-input/32 [&>svg]:text-muted-foreground
variant.error: border-destructive/32 bg-destructive/4 [&>svg]:text-destructive
variant.info: border-info/32 bg-info/4 [&>svg]:text-info
variant.success: border-success/32 bg-success/4 [&>svg]:text-success
variant.warning: border-warning/32 bg-warning/4 [&>svg]:text-warning
```

## Parts

### `Alert` → `[data-slot=alert]` · `<div>` (line 32)

Attributes: `role=alert`

#### base (cva defaults: variant=default)

```txt
relative grid w-full items-start gap-x-2 gap-y-0.5 rounded-xl border px-3.5 py-3 text-card-foreground text-sm has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto] has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr] has-data-[slot=alert-action]:grid-cols-[1fr_auto] has-[>svg]:gap-x-2 [&>svg]:h-lh [&>svg]:w-4 bg-transparent dark:bg-input/32 [&>svg]:text-muted-foreground
```

**element — default**

Key metrics: display `grid` · width `100%` · padding-inline `14px` · padding-block `12px` · column-gap `8px` · row-gap `2px` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `14px` · background-color `#00000000` / `#ffffff07` · color `#262626` / `#f5f5f5` (card-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `grid` | `grid` | `grid` |
| width | `100%` | `100%` | `w-full` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| column-gap | `8px` | `calc(var(--spacing) * 2)` | `gap-x-2` |
| row-gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-y-0.5` |
| border-radius | `14px` | `calc(var(--radius) + 4px)` | `rounded-xl` |
| background-color | `#00000000` / `#ffffff07` | `transparent` | `bg-transparent` |
| padding-inline | `14px` | `calc(var(--spacing) * 3.5)` | `px-3.5` |
| padding-block | `12px` | `calc(var(--spacing) * 3)` | `py-3` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#262626` / `#f5f5f5` | `card-foreground` | `text-card-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>svg) | grid-template-columns | `16px 1fr` | `has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr]` |
| has([data-slot=alert-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=alert-action]:grid-cols-[1fr_auto]` |
| has(>svg) + has([data-slot=alert-action]) | grid-template-columns | `16px 1fr auto` | `has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto]` |

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `1lh` | `1lh` | `[&>svg]:h-lh` |
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:w-4` |
| color | `#686868` / `#818181` | `muted-foreground` | `[&>svg]:text-muted-foreground` |

#### alertVariants: variant=error

```txt
border-destructive/32 bg-destructive/4 [&>svg]:text-destructive
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#fb2c3652` / `#fb414a52` | `destructive/32%` | `border-destructive/32` |
| background-color | `#fb2c360a` / `#fb414a0a` | `destructive/4%` | `bg-destructive/4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>svg) | grid-template-columns | `16px 1fr` | `has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr]` |
| has([data-slot=alert-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=alert-action]:grid-cols-[1fr_auto]` |
| has(>svg) + has([data-slot=alert-action]) | grid-template-columns | `16px 1fr auto` | `has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto]` |

**`>svg` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#fb2c36` / `#fb414a` | `destructive` | `[&>svg]:text-destructive` |

#### alertVariants: variant=info

```txt
border-info/32 bg-info/4 [&>svg]:text-info
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#2b7fff52` | `info/32%` | `border-info/32` |
| background-color | `#2b7fff0a` | `info/4%` | `bg-info/4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>svg) | grid-template-columns | `16px 1fr` | `has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr]` |
| has([data-slot=alert-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=alert-action]:grid-cols-[1fr_auto]` |
| has(>svg) + has([data-slot=alert-action]) | grid-template-columns | `16px 1fr auto` | `has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto]` |

**`>svg` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#2b7fff` | `info` | `[&>svg]:text-info` |

#### alertVariants: variant=success

```txt
border-success/32 bg-success/4 [&>svg]:text-success
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#00bc7d52` | `success/32%` | `border-success/32` |
| background-color | `#00bc7d0a` | `success/4%` | `bg-success/4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>svg) | grid-template-columns | `16px 1fr` | `has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr]` |
| has([data-slot=alert-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=alert-action]:grid-cols-[1fr_auto]` |
| has(>svg) + has([data-slot=alert-action]) | grid-template-columns | `16px 1fr auto` | `has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto]` |

**`>svg` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#00bc7d` | `success` | `[&>svg]:text-success` |

#### alertVariants: variant=warning

```txt
border-warning/32 bg-warning/4 [&>svg]:text-warning
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#fe9a0052` | `warning/32%` | `border-warning/32` |
| background-color | `#fe9a000a` | `warning/4%` | `bg-warning/4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>svg) | grid-template-columns | `16px 1fr` | `has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr]` |
| has([data-slot=alert-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=alert-action]:grid-cols-[1fr_auto]` |
| has(>svg) + has([data-slot=alert-action]) | grid-template-columns | `16px 1fr auto` | `has-[>svg]:has-data-[slot=alert-action]:grid-cols-[calc(var(--spacing)*4)_1fr_auto]` |

**`>svg` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#fe9a00` | `warning` | `[&>svg]:text-warning` |

### `AlertTitle` → `[data-slot=alert-title]` · `<div>` (line 46)

#### base

```txt
font-medium [svg~&]:col-start-2
```

**element — default**

Key metrics: font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `500` | `font-weight-medium` | `font-medium` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prevsib(svg) | grid-column-start | `2` | `[svg~&]:col-start-2` |

### `AlertDescription` → `[data-slot=alert-description]` · `<div>` (line 59)

#### base

```txt
flex flex-col gap-2.5 text-muted-foreground [svg~&]:col-start-2
```

**element — default**

Key metrics: display `flex` · gap `10px` · color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `10px` | `calc(var(--spacing) * 2.5)` | `gap-2.5` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prevsib(svg) | grid-column-start | `2` | `[svg~&]:col-start-2` |

### `AlertAction` → `[data-slot=alert-action]` · `<div>` (line 75)

#### base

```txt
flex gap-1 max-sm:col-start-2 max-sm:mt-2 sm:row-start-1 sm:row-end-3 sm:self-center sm:[[data-slot=alert-description]~&]:col-start-2 sm:[[data-slot=alert-title]~&]:col-start-2 sm:[svg~&]:col-start-2 sm:[svg~[data-slot=alert-description]~&]:col-start-3 sm:[svg~[data-slot=alert-title]~&]:col-start-3
```

**element — default**

Key metrics: display `flex` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| grid-row-start | `1` | `1` | `sm:row-start-1` |
| grid-row-end | `3` | `3` | `sm:row-end-3` |
| align-self | `center` | `center` | `sm:self-center` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prevsib([data-slot=alert-description]) | grid-column-start | `2` | `sm:[[data-slot=alert-description]~&]:col-start-2` |
| prevsib([data-slot=alert-title]) | grid-column-start | `2` | `sm:[[data-slot=alert-title]~&]:col-start-2` |
| prevsib(svg) | grid-column-start | `2` | `sm:[svg~&]:col-start-2` |
| prevsib(svg~[data-slot=alert-description]) | grid-column-start | `3` | `sm:[svg~[data-slot=alert-description]~&]:col-start-3` |
| prevsib(svg~[data-slot=alert-title]) | grid-column-start | `3` | `sm:[svg~[data-slot=alert-title]~&]:col-start-3` |

**element — mobile differences**: grid-row-start `(unset)` · grid-row-end `(unset)` · align-self `(unset)` · grid-column-start `2` · margin-top `8px`

