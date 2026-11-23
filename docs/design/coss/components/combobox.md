# Combobox

Source: `apps/ui/registry/default/ui/combobox.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/combobox.mdx` · Base UI: https://base-ui.com/react/components/combobox#api-reference

> An input combined with a list of predefined items to select.

Depends on: `@base-ui/react/combobox`, `lucide-react`, `@/ui/input`, `@/ui/scroll-area`

Exports: `Combobox`, `ComboboxChipsInput`, `ComboboxInput`, `ComboboxTrigger`, `ComboboxPopup`, `ComboboxItem`, `ComboboxSeparator`, `ComboboxGroup`, `ComboboxGroupLabel`, `ComboboxEmpty`, `ComboboxRow`, `ComboboxList`, `ComboboxClear`, `ComboboxStatus`, `ComboboxChips`, `ComboboxChip`, `ComboboxChipRemove`

## Behavior & keyboard

- Input + listbox (single or multiple with chips). Typing filters, `ArrowDown/Up` highlight, `Enter` selects, `Escape` closes; in multiple mode `Backspace` in empty input removes last chip, `ArrowLeft/Right` move between chips.
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Combobox`

No own classes (re-export / wrapper of the primitive).

### `ComboboxChipsInput` → `[data-slot=combobox-chips-input]` · `<ComboboxPrimitive.Input>` (line 47)

Attributes: `data-size={typeof sizeValue === "string" ? sizeValue : undefined}`, `size={typeof sizeValue === "number" ? sizeValue : undefined}`

#### base

```txt
min-w-12 flex-1 text-base text-foreground outline-none sm:text-sm [[data-slot=combobox-chip]+&]:ps-0.5
```

**element — default**

Key metrics: min-width `48px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-width | `48px` | `calc(var(--spacing) * 12)` | `min-w-12` |
| flex | `1` | `1` | `flex-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prev([data-slot=combobox-chip]) | padding-inline-start | `2px` | `[[data-slot=combobox-chip]+&]:ps-0.5` |

**element — mobile differences**: font-size `16px` · line-height `24px`

#### when `sizeValue === "sm"`

```txt
ps-1.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-start | `6px` | `calc(var(--spacing) * 1.5)` | `ps-1.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prev([data-slot=combobox-chip]) | padding-inline-start | `2px` | `[[data-slot=combobox-chip]+&]:ps-0.5` |

**element — mobile differences**: font-size `16px` · line-height `24px`

#### when `!(sizeValue === "sm")`

```txt
ps-2
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-start | `8px` | `calc(var(--spacing) * 2)` | `ps-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prev([data-slot=combobox-chip]) | padding-inline-start | `2px` | `[[data-slot=combobox-chip]+&]:ps-0.5` |

**element — mobile differences**: font-size `16px` · line-height `24px`

### `ComboboxInput` → `[data-slot=combobox-input-group]` · `<ComboboxPrimitive.InputGroup>` (line 82)

Prop defaults: `showTrigger=true`, `showClear=false`

#### base

```txt
relative not-has-[>*.w-full]:w-fit w-full text-foreground has-disabled:opacity-64
```

**element — default**

Key metrics: width `fit-content` · color `#262626` / `#f5f5f5` (foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `fit-content` | `fit-content` | `not-has-[>*.w-full]:w-fit` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>*.w-full) | width | `100%` | `w-full` |
| disabled | opacity | `64%` | `has-disabled:opacity-64` |

### `ComboboxInput` → `[data-slot=combobox-start-addon]` · `<div>` (line 87)

Attributes: `aria-hidden=true`

Prop defaults: `showTrigger=true`, `showClear=false`

#### base

```txt
pointer-events-none absolute inset-y-0 start-px z-10 flex items-center ps-[calc(--spacing(3)-1px)] opacity-80 has-[+[data-size=sm]]:ps-[calc(--spacing(2.5)-1px)] [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:-mx-0.5
```

**element — default**

Key metrics: display `flex` · padding-inline-start `11px` · opacity `80%` · z-index `10`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `inset-y-0` |
| inset-inline-start | `1px` | `1px` | `start-px` |
| z-index | `10` | `10` | `z-10` |
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| padding-inline-start | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `ps-[calc(--spacing(3)-1px)]` |
| opacity | `80%` | `80%` | `opacity-80` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-size=sm]) | padding-inline-start | `9px` | `has-[+[data-size=sm]]:ps-[calc(--spacing(2.5)-1px)]` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-mx-0.5` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ComboboxInput` → `[data-slot=combobox-input]` · `<ComboboxPrimitive.Input>` (line 95)

Attributes: `render={<Input className="has-disabled:opacity-100" nativeInput size={sizeValue} />}`

Prop defaults: `showTrigger=true`, `showClear=false`

#### when `startAddon`

```txt
data-[size=sm]:*:data-[slot=combobox-input]:ps-[calc(--spacing(7.5)-1px)] *:data-[slot=combobox-input]:ps-[calc(--spacing(8.5)-1px)] sm:data-[size=sm]:*:data-[slot=combobox-input]:ps-[calc(--spacing(7)-1px)] sm:*:data-[slot=combobox-input]:ps-[calc(--spacing(8)-1px)]
```

**`> *[data-slot="combobox-input"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-start | `31px` | `calc(calc(var(--spacing) * 8) - 1px)` | `sm:*:data-[slot=combobox-input]:ps-[calc(--spacing(8)-1px)]` |

**`> *[data-slot="combobox-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-size=sm] | padding-inline-start | `27px` | `sm:data-[size=sm]:*:data-[slot=combobox-input]:ps-[calc(--spacing(7)-1px)]` |

**`> *[data-slot="combobox-input"]` — mobile differences**: padding-inline-start `33px`

#### when `sizeValue === "sm"`

```txt
has-[+[data-slot=combobox-trigger],+[data-slot=combobox-clear]]:*:data-[slot=combobox-input]:pe-6.5
```

**`> *[data-slot="combobox-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-slot=combobox-trigger]) | padding-inline-end | `26px` | `has-[+[data-slot=combobox-trigger],+[data-slot=combobox-clear]]:*:data-[slot=combobox-input]:pe-6.5` |
| has(+[data-slot=combobox-clear]) | padding-inline-end | `26px` | `has-[+[data-slot=combobox-trigger],+[data-slot=combobox-clear]]:*:data-[slot=combobox-input]:pe-6.5` |

#### when `!(sizeValue === "sm")`

```txt
has-[+[data-slot=combobox-trigger],+[data-slot=combobox-clear]]:*:data-[slot=combobox-input]:pe-7
```

**`> *[data-slot="combobox-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-slot=combobox-trigger]) | padding-inline-end | `28px` | `has-[+[data-slot=combobox-trigger],+[data-slot=combobox-clear]]:*:data-[slot=combobox-input]:pe-7` |
| has(+[data-slot=combobox-clear]) | padding-inline-end | `28px` | `has-[+[data-slot=combobox-trigger],+[data-slot=combobox-clear]]:*:data-[slot=combobox-input]:pe-7` |

### `ComboboxInput` → (no data-slot) · `<Input>` (line 106)

Attributes: `nativeInput=true`, `size={sizeValue}`

Prop defaults: `showTrigger=true`, `showClear=false`

#### base

```txt
has-disabled:opacity-100
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | opacity | `100%` | `has-disabled:opacity-100` |

### `ComboboxInput` → (no data-slot) · `<ComboboxTrigger>` (line 115)

Prop defaults: `showTrigger=true`, `showClear=false`

#### base

```txt
absolute top-1/2 inline-flex size-8 shrink-0 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md border border-transparent opacity-80 outline-none transition-opacity pointer-coarse:after:absolute pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:opacity-100 has-[+[data-slot=combobox-clear]]:hidden sm:size-7 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · height `28px` · width `28px` · border-width `1px` · border-color `#00000000` · border-radius `8px` · opacity `80%` · transition-property `opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| position | `absolute` | `absolute` | `absolute` |
| top | `50%` | `calc(1/2 * 100%)` | `top-1/2` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| width | `28px` | `calc(var(--spacing) * 7)` | `sm:size-7` |
| height | `28px` | `calc(var(--spacing) * 7)` | `sm:size-7` |
| flex-shrink | `0` | `0` | `shrink-0` |
| translate | `0 -50%` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-y-1/2` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| opacity | `80%` | `80%` | `opacity-80` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-opacity` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=combobox-clear]) | display | `none` | `has-[+[data-slot=combobox-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

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

#### when `sizeValue === "sm"`

```txt
end-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-end | `0px` | `calc(var(--spacing) * 0)` | `end-0` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=combobox-clear]) | display | `none` | `has-[+[data-slot=combobox-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### when `!(sizeValue === "sm")`

```txt
end-0.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-end | `2px` | `calc(var(--spacing) * 0.5)` | `end-0.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=combobox-clear]) | display | `none` | `has-[+[data-slot=combobox-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ComboboxInput` → `[data-slot=combobox-icon]` · `<ComboboxPrimitive.Icon>` (line 122)

Prop defaults: `showTrigger=true`, `showClear=false`

### `ComboboxInput` → (no data-slot) · `<ComboboxClear>` (line 128)

Prop defaults: `showTrigger=true`, `showClear=false`

#### base

```txt
absolute top-1/2 inline-flex size-8 shrink-0 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md border border-transparent opacity-80 outline-none transition-opacity pointer-coarse:after:absolute pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:opacity-100 has-[+[data-slot=combobox-clear]]:hidden sm:size-7 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · height `28px` · width `28px` · border-width `1px` · border-color `#00000000` · border-radius `8px` · opacity `80%` · transition-property `opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| position | `absolute` | `absolute` | `absolute` |
| top | `50%` | `calc(1/2 * 100%)` | `top-1/2` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| width | `28px` | `calc(var(--spacing) * 7)` | `sm:size-7` |
| height | `28px` | `calc(var(--spacing) * 7)` | `sm:size-7` |
| flex-shrink | `0` | `0` | `shrink-0` |
| translate | `0 -50%` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-y-1/2` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| opacity | `80%` | `80%` | `opacity-80` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-opacity` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=combobox-clear]) | display | `none` | `has-[+[data-slot=combobox-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

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

#### when `sizeValue === "sm"`

```txt
end-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-end | `0px` | `calc(var(--spacing) * 0)` | `end-0` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=combobox-clear]) | display | `none` | `has-[+[data-slot=combobox-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### when `!(sizeValue === "sm")`

```txt
end-0.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-end | `2px` | `calc(var(--spacing) * 0.5)` | `end-0.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=combobox-clear]) | display | `none` | `has-[+[data-slot=combobox-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ComboboxTrigger` → `[data-slot=combobox-trigger]` · `<ComboboxPrimitive.Trigger>` (line 148)

### `ComboboxPopup` → `[data-slot=combobox-positioner]` · `<ComboboxPrimitive.Positioner>` (line 181)

Attributes: `align={align}`, `alignOffset={alignOffset}`, `anchor={anchor}`, `side={side}`, `sideOffset={sideOffset}`

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`

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

### `ComboboxPopup` → (no data-slot) · `<span>` (line 190)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`

#### base

```txt
relative flex max-h-full min-w-(--anchor-width) max-w-(--available-width) origin-(--transform-origin) rounded-lg border bg-popover not-dark:bg-clip-padding shadow-lg/5 transition-[scale,opacity] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · max-height `100%` · min-width `var(--anchor-width)` (runtime var --anchor-width) · max-width `var(--available-width)` (runtime var --available-width) · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `#ffffff` / `#1d1d1d` (popover) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · transition-property `scale, opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| max-height | `100%` | `100%` | `max-h-full` |
| max-width | `var(--available-width)` | `runtime var --available-width` | `max-w-(--available-width)` |
| min-width | `var(--anchor-width)` | `runtime var --anchor-width` | `min-w-(--anchor-width)` |
| transform-origin | `var(--transform-origin)` | `runtime var --transform-origin` | `origin-(--transform-origin)` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `scale, opacity` | `scale,opacity` | `transition-[scale,opacity]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[scale,opacity]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[scale,opacity]` |
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

### `ComboboxPopup` → `[data-slot=combobox-popup]` · `<ComboboxPrimitive.Popup>` (line 196)

Prop defaults: `side="bottom"`, `sideOffset=4`, `align="start"`

#### base

```txt
flex max-h-[min(var(--available-height),23rem)] flex-1 flex-col text-foreground
```

**element — default**

Key metrics: display `flex` · max-height `min(var(--available-height), 368px)` · color `#262626` / `#f5f5f5` (foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| max-height | `min(var(--available-height), 368px)` | `min(var(--available-height), 23rem)` | `max-h-[min(var(--available-height),23rem)]` |
| flex | `1` | `1` | `flex-1` |
| flex-direction | `column` | `column` | `flex-col` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

### `ComboboxItem` → `[data-slot=combobox-item]` · `<ComboboxPrimitive.Item>` (line 215)

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

### `ComboboxItem` → (no data-slot) · `<ComboboxPrimitive.ItemIndicator>` (line 223)

#### base

```txt
col-start-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |

### `ComboboxItem` → (no data-slot) · `<div>` (line 239)

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

### `ComboboxSeparator` → `[data-slot=combobox-separator]` · `<ComboboxPrimitive.Separator>` (line 249)

#### base

```txt
mx-2 my-1 h-px bg-border last:hidden
```

**element — default**

Key metrics: height `1px` · background-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `8px` | `calc(var(--spacing) * 2)` | `mx-2` |
| margin-block | `4px` | `calc(var(--spacing) * 1)` | `my-1` |
| height | `1px` | `1px` | `h-px` |
| background-color | `#00000014` / `#ffffff0f` | `border` | `bg-border` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :last-child | display | `none` | `last:hidden` |

### `ComboboxGroup` → `[data-slot=combobox-group]` · `<ComboboxPrimitive.Group>` (line 262)

#### base

```txt
[[role=group]+&]:mt-1.5
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prev([role=group]) | margin-top | `6px` | `[[role=group]+&]:mt-1.5` |

### `ComboboxGroupLabel` → `[data-slot=combobox-group-label]` · `<ComboboxPrimitive.GroupLabel>` (line 275)

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

### `ComboboxEmpty` → `[data-slot=combobox-empty]` · `<ComboboxPrimitive.Empty>` (line 291)

#### base

```txt
not-empty:p-2 text-center text-base text-muted-foreground sm:text-sm
```

**element — default**

Key metrics: padding `8px` · color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| text-align | `center` | `center` | `text-center` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |
| padding | `8px` | `calc(var(--spacing) * 2)` | `not-empty:p-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | padding | `(unset)` | `(removed)` |

**element — mobile differences**: font-size `16px` · line-height `24px`

### `ComboboxRow` → `[data-slot=combobox-row]` · `<ComboboxPrimitive.Row>` (line 307)

### `ComboboxList` → `[data-slot=combobox-list]` · `<ComboboxPrimitive.List>` (line 324)

#### base

```txt
not-empty:scroll-py-1 not-empty:px-1 not-empty:py-1 in-data-has-overflow-y:pe-3
```

**element — default**

Key metrics: padding-inline `4px` · padding-block `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| scroll-padding-block | `4px` | `calc(var(--spacing) * 1)` | `not-empty:scroll-py-1` |
| padding-inline | `4px` | `calc(var(--spacing) * 1)` | `not-empty:px-1` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `not-empty:py-1` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | scroll-padding-block | `(unset)` | `(removed)` |
| :empty | padding-inline | `(unset)` | `(removed)` |
| :empty | padding-block | `(unset)` | `(removed)` |
| in([data-has-overflow-y]) | padding-inline-end | `12px` | `in-data-has-overflow-y:pe-3` |

### `ComboboxClear` → `[data-slot=combobox-clear]` · `<ComboboxPrimitive.Clear>` (line 341)

### `ComboboxStatus` → `[data-slot=combobox-status]` · `<ComboboxPrimitive.Status>` (line 354)

#### base

```txt
px-3 py-2 font-medium text-muted-foreground text-xs empty:m-0 empty:p-0
```

**element — default**

Key metrics: padding-inline `12px` · padding-block `8px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `12px` | `calc(var(--spacing) * 3)` | `px-3` |
| padding-block | `8px` | `calc(var(--spacing) * 2)` | `py-2` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | margin | `0px` | `empty:m-0` |
| :empty | padding | `0px` | `empty:p-0` |

### `ComboboxChips` → `[data-slot=combobox-chips]` · `<ComboboxPrimitive.Chips>` (line 379)

#### base

```txt
relative inline-flex min-h-9 w-full flex-wrap gap-1 rounded-lg border border-input bg-background not-dark:bg-clip-padding p-[calc(--spacing(1)-1px)] text-base shadow-xs/5 outline-none ring-ring/24 transition-shadow *:min-h-7 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-has-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] focus-within:border-ring focus-within:ring-[3px] has-disabled:pointer-events-none has-data-[size=lg]:min-h-10 has-data-[size=sm]:min-h-8 has-aria-invalid:border-destructive/36 has-disabled:opacity-64 has-[:disabled,:focus-within,[aria-invalid]]:shadow-none focus-within:has-aria-invalid:border-destructive/64 focus-within:has-aria-invalid:ring-destructive/16 has-data-[size=lg]:*:min-h-8 has-data-[size=sm]:*:min-h-6 sm:min-h-8 sm:text-sm sm:has-data-[size=lg]:min-h-9 sm:has-data-[size=sm]:min-h-7 sm:*:min-h-6 sm:has-data-[size=lg]:*:min-h-7 sm:has-data-[size=sm]:*:min-h-5 dark:not-has-disabled:bg-input/32 dark:has-aria-invalid:ring-destructive/24 dark:not-has-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `inline-flex` · min-height `32px` · width `100%` · padding `3px` · gap `4px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#ffffff` / `#ffffff07` (background) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| flex-wrap | `wrap` | `wrap` | `flex-wrap` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| padding | `3px` | `calc(calc(var(--spacing) * 1) - 1px)` | `p-[calc(--spacing(1)-1px)]` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | background-color | `#ffffff` / `#141414` | `bg-background` |
| disabled | box-shadow | `none` | `has-[:disabled,:focus-within,[aria-invalid]]:shadow-none` |
| disabled | pointer-events | `none` | `has-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `has-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-within:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `has-[:disabled,:focus-within,[aria-invalid]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `has-aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `has-[:disabled,:focus-within,[aria-invalid]]:shadow-none` |
| has([data-size=lg]) | min-height | `36px` | `sm:has-data-[size=lg]:min-h-9` |
| has([data-size=sm]) | min-height | `28px` | `sm:has-data-[size=sm]:min-h-7` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-within:has-aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `has-[:disabled,:focus-within,[aria-invalid]]:shadow-none` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-has-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-has-disabled:not-focus-within:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`> *` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `24px` | `calc(var(--spacing) * 6)` | `sm:*:min-h-6` |

**`> *` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-size=lg]) | min-height | `28px` | `sm:has-data-[size=lg]:*:min-h-7` |
| has([data-size=sm]) | min-height | `20px` | `sm:has-data-[size=sm]:*:min-h-5` |

**`> *` — mobile differences**: min-height `28px`

### `ComboboxChips` → `[data-slot=combobox-start-addon]` · `<div>` (line 389)

Attributes: `aria-hidden=true`

#### base

```txt
flex shrink-0 items-center ps-2 opacity-80 has-[~[data-size=sm]]:has-[+[data-slot=combobox-chip]]:pe-1.5 has-[~[data-size=sm]]:ps-1.5 has-[+[data-slot=combobox-chip]]:pe-2 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-ms-0.5 [&_svg]:-me-1.5
```

**element — default**

Key metrics: display `flex` · padding-inline-start `8px` · opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| padding-inline-start | `8px` | `calc(var(--spacing) * 2)` | `ps-2` |
| opacity | `80%` | `80%` | `opacity-80` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(~[data-size=sm]) | padding-inline-start | `6px` | `has-[~[data-size=sm]]:ps-1.5` |
| has(+[data-slot=combobox-chip]) | padding-inline-end | `8px` | `has-[+[data-slot=combobox-chip]]:pe-2` |
| has(+[data-slot=combobox-chip]) + has(~[data-size=sm]) | padding-inline-start | `6px` | `has-[~[data-size=sm]]:ps-1.5` |
| has(+[data-slot=combobox-chip]) + has(~[data-size=sm]) | padding-inline-end | `6px` | `has-[~[data-size=sm]]:has-[+[data-slot=combobox-chip]]:pe-1.5` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| margin-inline-start | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-ms-0.5` |
| margin-inline-end | `-6px` | `calc(var(--spacing) * -1.5)` | `[&_svg]:-me-1.5` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ComboboxChip` → `[data-slot=combobox-chip]` · `<ComboboxPrimitive.Chip>` (line 410)

#### base

```txt
flex items-center rounded-[calc(var(--radius-md)-1px)] bg-accent ps-2 font-medium text-accent-foreground text-sm outline-none sm:text-xs/(--text-xs--line-height) [&_svg:not([class*='size-'])]:size-4 sm:[&_svg:not([class*='size-'])]:size-3.5
```

**element — default**

Key metrics: display `flex` · padding-inline-start `8px` · border-radius `7px` · background-color `#0000000a` / `#ffffff0a` (accent) · color `#262626` / `#f5f5f5` (accent-foreground) · font-size `12px` (text-xs) · line-height `16px` (text-xs--line-height) · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `rounded-[calc(var(--radius-md)-1px)]` |
| background-color | `#0000000a` / `#ffffff0a` | `accent` | `bg-accent` |
| padding-inline-start | `8px` | `calc(var(--spacing) * 2)` | `ps-2` |
| font-size | `12px` | `text-xs` | `sm:text-xs/(--text-xs--line-height)` |
| line-height | `16px` | `text-xs--line-height` | `sm:text-xs/(--text-xs--line-height)` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `accent-foreground` | `text-accent-foreground` |
| outline-style | `none` | `none` | `outline-none` |

**element — mobile differences**: font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `14px` | `calc(var(--spacing) * 3.5)` | `sm:[&_svg:not([class*='size-'])]:size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `sm:[&_svg:not([class*='size-'])]:size-3.5` |

**`svg:not([class*='size-'])` — mobile differences**: width `16px` · height `16px`

### `ComboboxChipRemove` → `[data-slot=combobox-chip-remove]` · `<ComboboxPrimitive.ChipRemove>` (line 425)

#### base

```txt
h-full shrink-0 cursor-pointer px-1.5 opacity-80 hover:opacity-100 [&_svg:not([class*='size-'])]:size-4 sm:[&_svg:not([class*='size-'])]:size-3.5
```

**element — default**

Key metrics: height `100%` · padding-inline `6px` · opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `100%` | `100%` | `h-full` |
| flex-shrink | `0` | `0` | `shrink-0` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| padding-inline | `6px` | `calc(var(--spacing) * 1.5)` | `px-1.5` |
| opacity | `80%` | `80%` | `opacity-80` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `14px` | `calc(var(--spacing) * 3.5)` | `sm:[&_svg:not([class*='size-'])]:size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `sm:[&_svg:not([class*='size-'])]:size-3.5` |

**`svg:not([class*='size-'])` — mobile differences**: width `16px` · height `16px`

