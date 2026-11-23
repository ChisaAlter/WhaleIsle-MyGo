# Select

Source: `apps/ui/registry/default/ui/select.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/select.mdx` · Base UI: https://base-ui.com/react/components/select#api-reference

> A common form component for choosing a predefined value in a dropdown menu.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/select`, `@base-ui/react/use-render`, `lucide-react`

Exports: `SelectButton`, `SelectTrigger`, `SelectValue`, `SelectPopup`, `SelectItem`, `SelectSeparator`, `SelectGroup`, `SelectLabel`, `SelectGroupLabel`

Module constants: `selectTriggerIconClassName = -me-1 size-4.5 opacity-80 sm:size-4`

## Behavior & keyboard

- Trigger opens with click/`Enter`/`Space`/`ArrowDown`; popup aligns selected item over trigger by default (`alignItemWithTrigger`); `ArrowUp/Down` highlight, typeahead, `Enter` selects and closes, `Escape` closes. Scroll arrows appear when list overflows.
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `selectTriggerVariants` (line 17)

Defaults: {"size":"default"}

```txt
base: relative inline-flex min-h-9 w-full min-w-36 select-none items-center justify-between gap-2 rounded-lg border border-input bg-background not-dark:bg-clip-padding px-[calc(--spacing(3)-1px)] text-left text-base text-foreground shadow-xs/5 outline-none ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 focus-visible:border-ring focus-visible:ring-[3px] aria-invalid:border-destructive/36 focus-visible:aria-invalid:border-destructive/64 focus-visible:aria-invalid:ring-destructive/16 data-disabled:pointer-events-none data-disabled:opacity-64 sm:min-h-8 sm:text-sm dark:bg-input/32 dark:aria-invalid:ring-destructive/24 dark:not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 [[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none
size.default: 
size.lg: min-h-10 sm:min-h-9
size.sm: min-h-8 gap-1.5 px-[calc(--spacing(2.5)-1px)] sm:min-h-7
```

## Parts

### `SelectButton` → `[data-slot=select-button]` · `useRender <button>` (line 49)

Attributes: `type={typeValue}`

#### base (cva defaults: size=default)

```txt
relative inline-flex min-h-9 w-full min-w-36 select-none items-center justify-between gap-2 rounded-lg border border-input bg-background not-dark:bg-clip-padding px-[calc(--spacing(3)-1px)] text-left text-base text-foreground shadow-xs/5 outline-none ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 focus-visible:border-ring focus-visible:ring-[3px] aria-invalid:border-destructive/36 focus-visible:aria-invalid:border-destructive/64 focus-visible:aria-invalid:ring-destructive/16 data-disabled:pointer-events-none data-disabled:opacity-64 sm:min-h-8 sm:text-sm dark:bg-input/32 dark:aria-invalid:ring-destructive/24 dark:not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 [[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none min-w-0
```

**element — default**

Key metrics: display `inline-flex` · min-height `32px` · width `100%` · min-width `144px` · padding-inline `11px` · gap `8px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#ffffff` / `#ffffff07` (background) · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| min-width | `144px` | `calc(var(--spacing) * 36)` | `min-w-36` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| text-align | `left` | `left` | `text-left` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| pressed | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='opacity-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `[&_svg:not([class*='opacity-'])]:opacity-80` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### selectTriggerVariants: size=lg

```txt
min-h-10 sm:min-h-9
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `36px` | `calc(var(--spacing) * 9)` | `sm:min-h-9` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| pressed | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |

**element — mobile differences**: min-height `40px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px`

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### selectTriggerVariants: size=sm

```txt
min-h-8 gap-1.5 px-[calc(--spacing(2.5)-1px)] sm:min-h-7
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| pressed | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px`

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `SelectButton` → (no data-slot) · `<span>` (line 52)

#### base

```txt
flex-1 truncate in-data-placeholder:text-muted-foreground/72
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex | `1` | `1` | `flex-1` |
| overflow | `hidden` | `hidden` | `truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `truncate` |
| white-space | `nowrap` | `nowrap` | `truncate` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-placeholder]) | color | `#686868b8` / `#818181b8` | `in-data-placeholder:text-muted-foreground/72` |

### `SelectButton` → (no data-slot) · `<ChevronsUpDownIcon>` (line 55)

#### base

```txt
-me-1 size-4.5 opacity-80 sm:size-4
```

**element — default**

Key metrics: height `16px` · width `16px` · opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-end | `-4px` | `calc(var(--spacing) * -1)` | `-me-1` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| opacity | `80%` | `80%` | `opacity-80` |

**element — mobile differences**: width `18px` · height `18px`

### `SelectTrigger` → `[data-slot=select-trigger]` · `<SelectPrimitive.Trigger>` (line 78)

Prop defaults: `size="default"`

#### base (cva defaults: size=default)

```txt
relative inline-flex min-h-9 w-full min-w-36 select-none items-center justify-between gap-2 rounded-lg border border-input bg-background not-dark:bg-clip-padding px-[calc(--spacing(3)-1px)] text-left text-base text-foreground shadow-xs/5 outline-none ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 focus-visible:border-ring focus-visible:ring-[3px] aria-invalid:border-destructive/36 focus-visible:aria-invalid:border-destructive/64 focus-visible:aria-invalid:ring-destructive/16 data-disabled:pointer-events-none data-disabled:opacity-64 sm:min-h-8 sm:text-sm dark:bg-input/32 dark:aria-invalid:ring-destructive/24 dark:not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 [[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none
```

**element — default**

Key metrics: display `inline-flex` · min-height `32px` · width `100%` · min-width `144px` · padding-inline `11px` · gap `8px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#ffffff` / `#ffffff07` (background) · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| min-width | `144px` | `calc(var(--spacing) * 36)` | `min-w-36` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| text-align | `left` | `left` | `text-left` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| pressed | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-data-disabled:not-focus-visible:not-aria-invalid:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='opacity-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `[&_svg:not([class*='opacity-'])]:opacity-80` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### selectTriggerVariants: size=lg

```txt
min-h-10 sm:min-h-9
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `36px` | `calc(var(--spacing) * 9)` | `sm:min-h-9` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| pressed | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |

**element — mobile differences**: min-height `40px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px`

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### selectTriggerVariants: size=sm

```txt
min-h-8 gap-1.5 px-[calc(--spacing(2.5)-1px)] sm:min-h-7
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| pressed | box-shadow | `none` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `[[data-disabled],:focus-visible,[aria-invalid],[data-pressed]]:shadow-none` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px`

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `SelectTrigger` → `[data-slot=select-icon]` · `<SelectPrimitive.Icon>` (line 84)

Prop defaults: `size="default"`

### `SelectTrigger` → (no data-slot) · `<ChevronsUpDownIcon>` (line 85)

Prop defaults: `size="default"`

#### base

```txt
-me-1 size-4.5 opacity-80 sm:size-4
```

**element — default**

Key metrics: height `16px` · width `16px` · opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-end | `-4px` | `calc(var(--spacing) * -1)` | `-me-1` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| opacity | `80%` | `80%` | `opacity-80` |

**element — mobile differences**: width `18px` · height `18px`

### `SelectValue` → `[data-slot=select-value]` · `<SelectPrimitive.Value>` (line 96)

#### base

```txt
flex-1 truncate data-placeholder:text-muted-foreground
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex | `1` | `1` | `flex-1` |
| overflow | `hidden` | `hidden` | `truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `truncate` |
| white-space | `nowrap` | `nowrap` | `truncate` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| placeholder | color | `#686868` / `#818181` | `data-placeholder:text-muted-foreground` |

### `SelectPopup` → `[data-slot=select-positioner]` · `<SelectPrimitive.Positioner>` (line 129)

Attributes: `align={align}`, `alignItemWithTrigger={alignItemWithTrigger}`, `alignOffset={alignOffset}`, `anchor={anchor}`, `side={side}`, `sideOffset={sideOffset}`

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
z-50 select-none
```

**element — default**

Key metrics: z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `50` | `50` | `z-50` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

### `SelectPopup` → `[data-slot=select-popup]` · `<SelectPrimitive.Popup>` (line 139)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
origin-(--transform-origin) text-foreground outline-none
```

**element — default**

Key metrics: color `#262626` / `#f5f5f5` (foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `var(--transform-origin)` | `runtime var --transform-origin` | `origin-(--transform-origin)` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |

### `SelectPopup` → `[data-slot=select-scroll-up-arrow]` · `<SelectPrimitive.ScrollUpArrow>` (line 144)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
top-0 z-50 flex h-6 w-full cursor-default items-center justify-center before:pointer-events-none before:absolute before:inset-x-px before:top-px before:h-[200%] before:rounded-t-[calc(var(--radius-lg)-1px)] before:bg-linear-to-b before:from-50% before:from-popover
```

**element — default**

Key metrics: display `flex` · height `24px` · width `100%` · z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| top | `0px` | `calc(var(--spacing) * 0)` | `top-0` |
| z-index | `50` | `50` | `z-50` |
| display | `flex` | `flex` | `flex` |
| height | `24px` | `calc(var(--spacing) * 6)` | `h-6` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:from-50%` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset-inline | `1px` | `1px` | `before:inset-x-px` |
| top | `1px` | `1px` | `before:top-px` |
| height | `200%` | `200%` | `before:h-[200%]` |
| border-top-left-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-t-[calc(var(--radius-lg)-1px)]` |
| border-top-right-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-t-[calc(var(--radius-lg)-1px)]` |
| background-image | `linear-gradient(to bottom in oklab, #fff 50%, #0000 100%)` / `linear-gradient(to bottom in oklab, #1d1d1d 50%, #0000 100%)` | `linear-gradient(var(--tw-gradient-stops))` | `before:bg-linear-to-b` |

### `SelectPopup` → (no data-slot) · `<ChevronUpIcon>` (line 148)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
relative size-4.5 sm:size-4
```

**element — default**

Key metrics: height `16px` · width `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |

**element — mobile differences**: width `18px` · height `18px`

### `SelectPopup` → (no data-slot) · `<div>` (line 150)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
relative h-full min-w-(--anchor-width) rounded-lg border bg-popover not-dark:bg-clip-padding shadow-lg/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: height `100%` · min-width `var(--anchor-width)` (runtime var --anchor-width) · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `#ffffff` / `#1d1d1d` (popover) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| height | `100%` | `100%` | `h-full` |
| min-width | `var(--anchor-width)` | `runtime var --anchor-width` | `min-w-(--anchor-width)` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `SelectPopup` → `[data-slot=select-list]` · `<SelectPrimitive.List>` (line 151)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
max-h-(--available-height) overflow-y-auto p-1
```

**element — default**

Key metrics: max-height `var(--available-height)` (runtime var --available-height) · padding `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| max-height | `var(--available-height)` | `runtime var --available-height` | `max-h-(--available-height)` |
| overflow-y | `auto` | `auto` | `overflow-y-auto` |
| padding | `4px` | `calc(var(--spacing) * 1)` | `p-1` |

### `SelectPopup` → `[data-slot=select-scroll-down-arrow]` · `<SelectPrimitive.ScrollDownArrow>` (line 161)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
bottom-0 z-50 flex h-6 w-full cursor-default items-center justify-center before:pointer-events-none before:absolute before:inset-x-px before:bottom-px before:h-[200%] before:rounded-b-[calc(var(--radius-lg)-1px)] before:bg-linear-to-t before:from-50% before:from-popover
```

**element — default**

Key metrics: display `flex` · height `24px` · width `100%` · z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| bottom | `0px` | `calc(var(--spacing) * 0)` | `bottom-0` |
| z-index | `50` | `50` | `z-50` |
| display | `flex` | `flex` | `flex` |
| height | `24px` | `calc(var(--spacing) * 6)` | `h-6` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:from-50%` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset-inline | `1px` | `1px` | `before:inset-x-px` |
| bottom | `1px` | `1px` | `before:bottom-px` |
| height | `200%` | `200%` | `before:h-[200%]` |
| border-bottom-right-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-b-[calc(var(--radius-lg)-1px)]` |
| border-bottom-left-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-b-[calc(var(--radius-lg)-1px)]` |
| background-image | `linear-gradient(to top in oklab, #fff 50%, #0000 100%)` / `linear-gradient(to top in oklab, #1d1d1d 50%, #0000 100%)` | `linear-gradient(var(--tw-gradient-stops))` | `before:bg-linear-to-t` |

### `SelectPopup` → (no data-slot) · `<ChevronDownIcon>` (line 165)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`, `alignOffset=0`, `alignItemWithTrigger=true`

#### base

```txt
relative size-4.5 sm:size-4
```

**element — default**

Key metrics: height `16px` · width `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |

**element — mobile differences**: width `18px` · height `18px`

### `SelectItem` → `[data-slot=select-item]` · `<SelectPrimitive.Item>` (line 179)

#### base

```txt
grid min-h-8 in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)] cursor-default grid-cols-[1rem_minmax(0,1fr)] items-center gap-2 rounded-sm py-1 ps-2 pe-4 text-base outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `grid` · min-height `28px` · padding-block `4px` · padding-inline-start `8px` · padding-inline-end `16px` · gap `8px` · border-radius `6px` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| cursor | `default` | `default` | `cursor-default` |
| grid-template-columns | `16px minmax(0,1fr)` | `1rem minmax(0,1fr)` | `grid-cols-[1rem_minmax(0,1fr)]` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| padding-inline-start | `8px` | `calc(var(--spacing) * 2)` | `ps-2` |
| padding-inline-end | `16px` | `calc(var(--spacing) * 4)` | `pe-4` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-side=none]) | min-width | `calc(var(--anchor-width) + 20px)` | `in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)]` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |
| highlighted | color | `#262626` / `#f5f5f5` | `data-highlighted:text-accent-foreground` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

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

### `SelectItem` → (no data-slot) · `<SelectPrimitive.ItemIndicator>` (line 187)

#### base

```txt
col-start-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |

### `SelectItem` → (no data-slot) · `<SelectPrimitive.ItemText>` (line 203)

#### base

```txt
wrap-anywhere col-start-2 min-w-0
```

**element — default**

Key metrics: min-width `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| overflow-wrap | `anywhere` | `anywhere` | `wrap-anywhere` |

### `SelectSeparator` → `[data-slot=select-separator]` · `<SelectPrimitive.Separator>` (line 215)

#### base

```txt
mx-2 my-1 h-px bg-border
```

**element — default**

Key metrics: height `1px` · background-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `8px` | `calc(var(--spacing) * 2)` | `mx-2` |
| margin-block | `4px` | `calc(var(--spacing) * 1)` | `my-1` |
| height | `1px` | `1px` | `h-px` |
| background-color | `#00000014` / `#ffffff0f` | `border` | `bg-border` |

### `SelectGroup` → `[data-slot=select-group]` · `<SelectPrimitive.Group>` (line 226)

### `SelectLabel` → `[data-slot=select-label]` · `<SelectPrimitive.Label>` (line 234)

#### base

```txt
not-in-data-[slot=field]:mb-2 inline-flex cursor-default items-center gap-2 font-medium text-base/4.5 text-foreground sm:text-sm/4
```

**element — default**

Key metrics: display `inline-flex` · gap `8px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| font-size | `14px` | `text-sm` | `sm:text-sm/4` |
| line-height | `16px` | `calc(var(--spacing) * 4)` | `sm:text-sm/4` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| margin-bottom | `8px` | `calc(var(--spacing) * 2)` | `not-in-data-[slot=field]:mb-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-slot=field]) | margin-bottom | `(unset)` | `(removed)` |

**element — mobile differences**: font-size `16px` · line-height `18px`

### `SelectGroupLabel` → `[data-slot=select-group-label]` · `<SelectPrimitive.GroupLabel>` (line 249)

#### base

```txt
px-2 py-1.5 font-medium text-muted-foreground text-xs
```

**element — default**

Key metrics: padding-inline `8px` · padding-block `6px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `6px` | `calc(var(--spacing) * 1.5)` | `py-1.5` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

