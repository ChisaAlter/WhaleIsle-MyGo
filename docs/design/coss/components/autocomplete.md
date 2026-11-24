# Autocomplete

Source: `apps/ui/registry/default/ui/autocomplete.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/autocomplete.mdx` · Base UI: https://base-ui.com/react/components/autocomplete#api-reference

> An input that suggests options as you type.

Depends on: `@base-ui/react/autocomplete`, `lucide-react`, `@/ui/input`, `@/ui/scroll-area`

Exports: `AutocompleteInput`, `AutocompletePopup`, `AutocompleteItem`, `AutocompleteSeparator`, `AutocompleteGroup`, `AutocompleteGroupLabel`, `AutocompleteEmpty`, `AutocompleteRow`, `AutocompleteList`, `AutocompleteClear`, `AutocompleteStatus`, `AutocompleteTrigger`

## Behavior & keyboard

- Input with listbox popup. Typing filters; `ArrowDown`/`ArrowUp` move the highlighted item (`data-highlighted`), `Enter` selects, `Escape` closes the popup (second `Escape` clears, per Base UI), `Home`/`End` move caret in input.
- Focus stays in the input (virtual focus via `aria-activedescendant`).
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

Docs mentions:

- Autofill the input with the highlighted item while navigating with arrow keys.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `AutocompleteInput` → `[data-slot=autocomplete-input-group]` · `<AutocompletePrimitive.InputGroup>` (line 34)

Prop defaults: `showTrigger=false`, `showClear=false`

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

### `AutocompleteInput` → `[data-slot=autocomplete-start-addon]` · `<div>` (line 39)

Attributes: `aria-hidden=true`

Prop defaults: `showTrigger=false`, `showClear=false`

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

### `AutocompleteInput` → `[data-slot=autocomplete-input]` · `<AutocompletePrimitive.Input>` (line 47)

Attributes: `render={<Input nativeInput size={sizeValue} />}`

Prop defaults: `showTrigger=false`, `showClear=false`

#### when `startAddon`

```txt
data-[size=sm]:*:data-[slot=autocomplete-input]:ps-[calc(--spacing(7.5)-1px)] *:data-[slot=autocomplete-input]:ps-[calc(--spacing(8.5)-1px)] sm:data-[size=sm]:*:data-[slot=autocomplete-input]:ps-[calc(--spacing(7)-1px)] sm:*:data-[slot=autocomplete-input]:ps-[calc(--spacing(8)-1px)]
```

**`> *[data-slot="autocomplete-input"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-start | `31px` | `calc(calc(var(--spacing) * 8) - 1px)` | `sm:*:data-[slot=autocomplete-input]:ps-[calc(--spacing(8)-1px)]` |

**`> *[data-slot="autocomplete-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-size=sm] | padding-inline-start | `27px` | `sm:data-[size=sm]:*:data-[slot=autocomplete-input]:ps-[calc(--spacing(7)-1px)]` |

**`> *[data-slot="autocomplete-input"]` — mobile differences**: padding-inline-start `33px`

#### when `sizeValue === "sm"`

```txt
has-[+[data-slot=autocomplete-trigger],+[data-slot=autocomplete-clear]]:*:data-[slot=autocomplete-input]:pe-6.5
```

**`> *[data-slot="autocomplete-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-slot=autocomplete-trigger]) | padding-inline-end | `26px` | `has-[+[data-slot=autocomplete-trigger],+[data-slot=autocomplete-clear]]:*:data-[slot=autocomplete-input]:pe-6.5` |
| has(+[data-slot=autocomplete-clear]) | padding-inline-end | `26px` | `has-[+[data-slot=autocomplete-trigger],+[data-slot=autocomplete-clear]]:*:data-[slot=autocomplete-input]:pe-6.5` |

#### when `!(sizeValue === "sm")`

```txt
has-[+[data-slot=autocomplete-trigger],+[data-slot=autocomplete-clear]]:*:data-[slot=autocomplete-input]:pe-7
```

**`> *[data-slot="autocomplete-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-slot=autocomplete-trigger]) | padding-inline-end | `28px` | `has-[+[data-slot=autocomplete-trigger],+[data-slot=autocomplete-clear]]:*:data-[slot=autocomplete-input]:pe-7` |
| has(+[data-slot=autocomplete-clear]) | padding-inline-end | `28px` | `has-[+[data-slot=autocomplete-trigger],+[data-slot=autocomplete-clear]]:*:data-[slot=autocomplete-input]:pe-7` |

### `AutocompleteInput` → (no data-slot) · `<AutocompleteTrigger>` (line 61)

Prop defaults: `showTrigger=false`, `showClear=false`

#### base

```txt
absolute top-1/2 inline-flex size-8 shrink-0 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md border border-transparent opacity-80 outline-none transition-colors pointer-coarse:after:absolute pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:opacity-100 has-[+[data-slot=autocomplete-clear]]:hidden sm:size-7 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · height `28px` · width `28px` · border-width `1px` · border-color `#00000000` · border-radius `8px` · opacity `80%` · transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

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
| transition-property | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from,` | `transition-colors` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-colors` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-colors` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=autocomplete-clear]) | display | `none` | `has-[+[data-slot=autocomplete-clear]]:hidden` |

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
| has(+[data-slot=autocomplete-clear]) | display | `none` | `has-[+[data-slot=autocomplete-clear]]:hidden` |

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
| has(+[data-slot=autocomplete-clear]) | display | `none` | `has-[+[data-slot=autocomplete-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `AutocompleteInput` → `[data-slot=autocomplete-icon]` · `<AutocompletePrimitive.Icon>` (line 68)

Prop defaults: `showTrigger=false`, `showClear=false`

### `AutocompleteInput` → (no data-slot) · `<AutocompleteClear>` (line 74)

Prop defaults: `showTrigger=false`, `showClear=false`

#### base

```txt
absolute top-1/2 inline-flex size-8 shrink-0 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md border border-transparent opacity-80 outline-none transition-colors pointer-coarse:after:absolute pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:opacity-100 has-[+[data-slot=autocomplete-clear]]:hidden sm:size-7 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · height `28px` · width `28px` · border-width `1px` · border-color `#00000000` · border-radius `8px` · opacity `80%` · transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

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
| transition-property | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from,` | `transition-colors` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-colors` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-colors` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |
| has(+[data-slot=autocomplete-clear]) | display | `none` | `has-[+[data-slot=autocomplete-clear]]:hidden` |

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
| has(+[data-slot=autocomplete-clear]) | display | `none` | `has-[+[data-slot=autocomplete-clear]]:hidden` |

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
| has(+[data-slot=autocomplete-clear]) | display | `none` | `has-[+[data-slot=autocomplete-clear]]:hidden` |

**element — mobile differences**: width `32px` · height `32px`

**`::after` — mobile differences**: content `""` · position `absolute` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `AutocompletePopup` → `[data-slot=autocomplete-positioner]` · `<AutocompletePrimitive.Positioner>` (line 108)

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

### `AutocompletePopup` → (no data-slot) · `<span>` (line 117)

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

### `AutocompletePopup` → `[data-slot=autocomplete-popup]` · `<AutocompletePrimitive.Popup>` (line 123)

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

### `AutocompleteItem` → `[data-slot=autocomplete-item]` · `<AutocompletePrimitive.Item>` (line 142)

#### base

```txt
flex min-h-8 cursor-default select-none items-center rounded-sm px-2 py-1 text-base outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm
```

**element — default**

Key metrics: display `flex` · min-height `28px` · padding-inline `8px` · padding-block `4px` · border-radius `6px` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |
| highlighted | color | `#262626` / `#f5f5f5` | `data-highlighted:text-accent-foreground` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

### `AutocompleteSeparator` → `[data-slot=autocomplete-separator]` · `<AutocompletePrimitive.Separator>` (line 160)

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

### `AutocompleteGroup` → `[data-slot=autocomplete-group]` · `<AutocompletePrimitive.Group>` (line 173)

#### base

```txt
[[role=group]+&]:mt-1.5
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prev([role=group]) | margin-top | `6px` | `[[role=group]+&]:mt-1.5` |

### `AutocompleteGroupLabel` → `[data-slot=autocomplete-group-label]` · `<AutocompletePrimitive.GroupLabel>` (line 186)

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

### `AutocompleteEmpty` → `[data-slot=autocomplete-empty]` · `<AutocompletePrimitive.Empty>` (line 202)

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

### `AutocompleteRow` → `[data-slot=autocomplete-row]` · `<AutocompletePrimitive.Row>` (line 218)

### `AutocompleteList` → `[data-slot=autocomplete-list]` · `<AutocompletePrimitive.List>` (line 235)

#### base

```txt
not-empty:scroll-py-1 not-empty:p-1 in-data-has-overflow-y:pe-3
```

**element — default**

Key metrics: padding `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| scroll-padding-block | `4px` | `calc(var(--spacing) * 1)` | `not-empty:scroll-py-1` |
| padding | `4px` | `calc(var(--spacing) * 1)` | `not-empty:p-1` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | scroll-padding-block | `(unset)` | `(removed)` |
| :empty | padding | `(unset)` | `(removed)` |
| in([data-has-overflow-y]) | padding-inline-end | `12px` | `in-data-has-overflow-y:pe-3` |

### `AutocompleteClear` → `[data-slot=autocomplete-clear]` · `<AutocompletePrimitive.Clear>` (line 252)

#### base

```txt
absolute end-0.5 top-1/2 inline-flex size-8 shrink-0 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md border border-transparent opacity-80 outline-none transition-[color,background-color,box-shadow,opacity] pointer-coarse:after:absolute pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:opacity-100 sm:size-7 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · height `28px` · width `28px` · border-width `1px` · border-color `#00000000` · border-radius `8px` · opacity `80%` · transition-property `color, background-color, box-shadow, opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| position | `absolute` | `absolute` | `absolute` |
| inset-inline-end | `2px` | `calc(var(--spacing) * 0.5)` | `end-0.5` |
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
| transition-property | `color, background-color, box-shadow, opacity` | `color,background-color,box-shadow,opacity` | `transition-[color,background-color,box-shadow,opacity]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[color,background-color,box-shadow,opacity]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[color,background-color,box-shadow,opacity]` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | opacity | `100%` | `hover:opacity-100` |

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

### `AutocompleteStatus` → `[data-slot=autocomplete-status]` · `<AutocompletePrimitive.Status>` (line 270)

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

### `AutocompleteTrigger` → `[data-slot=autocomplete-trigger]` · `<AutocompletePrimitive.Trigger>` (line 290)

