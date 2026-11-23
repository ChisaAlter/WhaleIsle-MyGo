# Number Field

Source: `apps/ui/registry/default/ui/number-field.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/number-field.mdx` · Base UI: https://base-ui.com/react/components/number-field#api-reference

> A numeric input element with increment and decrement buttons, and a scrub area.

Depends on: `@base-ui/react/number-field`, `lucide-react`, `@/ui/label`

Exports: `NumberField`, `NumberFieldGroup`, `NumberFieldDecrement`, `NumberFieldIncrement`, `NumberFieldInput`, `NumberFieldScrubArea`, `CursorGrowIcon`

## Behavior & keyboard

- Input with increment/decrement buttons. `ArrowUp`/`ArrowDown` step, `Shift+Arrow` large step, `PageUp/PageDown`, `Home/End` min/max; scrub area supports pointer drag. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `NumberField` → `[data-slot=number-field]` · `<NumberFieldPrimitive.Root>` (line 28)

Attributes: `data-size={size}`, `id={fieldId}`

Prop defaults: `size="default"`

#### base

```txt
flex w-full flex-col items-start gap-2
```

**element — default**

Key metrics: display `flex` · width `100%` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

### `NumberFieldGroup` → `[data-slot=number-field-group]` · `<NumberFieldPrimitive.Group>` (line 44)

#### base

```txt
relative flex w-full justify-between rounded-lg border border-input bg-background not-dark:bg-clip-padding text-base text-foreground shadow-xs/5 ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-data-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] focus-within:border-ring focus-within:ring-[3px] has-aria-invalid:border-destructive/36 focus-within:has-aria-invalid:border-destructive/64 focus-within:has-aria-invalid:ring-destructive/16 data-disabled:pointer-events-none data-disabled:opacity-64 sm:text-sm dark:bg-input/32 dark:has-aria-invalid:ring-destructive/24 dark:not-data-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 [[data-disabled],:focus-within,[aria-invalid]]:shadow-none
```

**element — default**

Key metrics: display `flex` · width `100%` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#ffffff` / `#ffffff07` (background) · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-within,[aria-invalid]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-within:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-within,[aria-invalid]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `has-aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-within,[aria-invalid]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-within:has-aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-within,[aria-invalid]]:shadow-none` |

**element — mobile differences**: font-size `16px` · line-height `24px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-data-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-data-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `NumberFieldDecrement` → `[data-slot=number-field-decrement]` · `<NumberFieldPrimitive.Decrement>` (line 60)

#### base

```txt
relative flex shrink-0 cursor-pointer items-center justify-center rounded-s-[calc(var(--radius-lg)-1px)] in-data-[size=sm]:px-[calc(--spacing(2.5)-1px)] px-[calc(--spacing(3)-1px)] transition-colors pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:bg-accent
```

**element — default**

Key metrics: display `flex` · padding-inline `11px` · transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-start-start-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `rounded-s-[calc(var(--radius-lg)-1px)]` |
| border-end-start-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `rounded-s-[calc(var(--radius-lg)-1px)]` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| transition-property | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from,` | `transition-colors` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-colors` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-colors` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-size=sm]) | padding-inline | `9px` | `in-data-[size=sm]:px-[calc(--spacing(2.5)-1px)]` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

### `NumberFieldIncrement` → `[data-slot=number-field-increment]` · `<NumberFieldPrimitive.Increment>` (line 78)

#### base

```txt
relative flex shrink-0 cursor-pointer items-center justify-center rounded-e-[calc(var(--radius-lg)-1px)] in-data-[size=sm]:px-[calc(--spacing(2.5)-1px)] px-[calc(--spacing(3)-1px)] transition-colors pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:bg-accent
```

**element — default**

Key metrics: display `flex` · padding-inline `11px` · transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-start-end-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `rounded-e-[calc(var(--radius-lg)-1px)]` |
| border-end-end-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `rounded-e-[calc(var(--radius-lg)-1px)]` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| transition-property | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from,` | `transition-colors` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-colors` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-colors` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-size=sm]) | padding-inline | `9px` | `in-data-[size=sm]:px-[calc(--spacing(2.5)-1px)]` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

### `NumberFieldInput` → `[data-slot=number-field-input]` · `<NumberFieldPrimitive.Input>` (line 96)

#### base

```txt
h-8.5 in-data-[size=lg]:h-9.5 in-data-[size=sm]:h-7.5 w-full min-w-0 grow bg-transparent in-data-[size=sm]:px-[calc(--spacing(2.5)-1px)] px-[calc(--spacing(3)-1px)] text-center text-foreground tabular-nums in-data-[size=lg]:leading-9.5 in-data-[size=sm]:leading-7.5 leading-8.5 outline-none sm:h-7.5 sm:in-data-[size=lg]:h-8.5 sm:in-data-[size=sm]:h-6.5 sm:in-data-[size=lg]:leading-8.5 sm:in-data-[size=sm]:leading-8.5 sm:leading-7.5
```

**element — default**

Key metrics: height `30px` · width `100%` · min-width `0px` · padding-inline `11px` · background-color `#00000000` · color `#262626` / `#f5f5f5` (foreground) · line-height `30px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `30px` | `calc(var(--spacing) * 7.5)` | `sm:h-7.5` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-grow | `1` | `1` | `grow` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| text-align | `center` | `center` | `text-center` |
| line-height | `30px` | `calc(var(--spacing) * 7.5)` | `sm:leading-7.5` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| font-variant-numeric | `tabular-nums` | `var(--tw-ordinal,) var(--tw-slashed-zero,) var(--tw-numeric-figure,) var(--tw-numeric-spacing,) var(--tw-numer` | `tabular-nums` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-size=lg]) | height | `34px` | `sm:in-data-[size=lg]:h-8.5` |
| in([data-size=lg]) | line-height | `34px` | `sm:in-data-[size=lg]:leading-8.5` |
| in([data-size=sm]) | height | `26px` | `sm:in-data-[size=sm]:h-6.5` |
| in([data-size=sm]) | padding-inline | `9px` | `in-data-[size=sm]:px-[calc(--spacing(2.5)-1px)]` |
| in([data-size=sm]) | line-height | `34px` | `sm:in-data-[size=sm]:leading-8.5` |

**element — mobile differences**: height `34px` · line-height `34px`

### `NumberFieldScrubArea` → `[data-slot=number-field-scrub-area]` · `<NumberFieldPrimitive.ScrubArea>` (line 123)

#### base

```txt
flex cursor-ew-resize
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| cursor | `ew-resize` | `ew-resize` | `cursor-ew-resize` |

### `NumberFieldScrubArea` → (no data-slot) · `<Label>` (line 128)

Attributes: `htmlFor={context.fieldId}`

#### base

```txt
cursor-ew-resize
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| cursor | `ew-resize` | `ew-resize` | `cursor-ew-resize` |

### `NumberFieldScrubArea` → (no data-slot) · `<NumberFieldPrimitive.ScrubAreaCursor>` (line 131)

#### base

```txt
drop-shadow-[0_1px_1px_#0008] filter
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| filter | `drop-shadow(0 1px 1px #0008)` | `var(--tw-blur,) var(--tw-brightness,) var(--tw-contrast,) var(--tw-grayscale,) var(--tw-hue-rotate,) var(--tw-` | `filter` |

### `CursorGrowIcon`

No own classes (re-export / wrapper of the primitive).

