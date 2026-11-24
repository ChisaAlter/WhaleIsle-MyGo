# Popover

Source: `apps/ui/registry/default/ui/popover.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/popover.mdx` · Base UI: https://base-ui.com/react/components/popover#api-reference

> An accessible popup anchored to a button.

Depends on: `@base-ui/react/popover`

Exports: `PopoverTrigger`, `PopoverPopup`, `PopoverClose`, `PopoverTitle`, `PopoverDescription`

## Behavior & keyboard

- Trigger toggles with click/`Enter`/`Space`; `Escape` and outside click close; focus moves into popup (non-modal by default) and returns to trigger.
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `PopoverTrigger` → `[data-slot=popover-trigger]` · `<PopoverPrimitive.Trigger>` (line 18)

### `PopoverPopup` → `[data-slot=popover-positioner]` · `<PopoverPrimitive.Positioner>` (line 50)

Attributes: `align={align}`, `alignOffset={alignOffset}`, `anchor={anchor}`, `side={side}`, `sideOffset={sideOffset}`

Prop defaults: `side="bottom"`, `align="center"`, `sideOffset=4`, `alignOffset=0`, `tooltipStyle=false`

#### base

```txt
z-50 h-(--positioner-height) w-(--positioner-width) max-w-(--available-width) transition-[top,left,right,bottom,transform] data-instant:transition-none
```

**element — default**

Key metrics: height `var(--positioner-height)` (runtime var --positioner-height) · width `var(--positioner-width)` (runtime var --positioner-width) · max-width `var(--available-width)` (runtime var --available-width) · z-index `50` · transition-property `top, left, right, bottom, transform` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `50` | `50` | `z-50` |
| height | `var(--positioner-height)` | `runtime var --positioner-height` | `h-(--positioner-height)` |
| width | `var(--positioner-width)` | `runtime var --positioner-width` | `w-(--positioner-width)` |
| max-width | `var(--available-width)` | `runtime var --available-width` | `max-w-(--available-width)` |
| transition-property | `top, left, right, bottom, transform` | `top,left,right,bottom,transform` | `transition-[top,left,right,bottom,transform]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[top,left,right,bottom,transform]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[top,left,right,bottom,transform]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-instant] | transition-property | `none` | `data-instant:transition-none` |

### `PopoverPopup` → `[data-slot=popover-popup]` · `<PopoverPrimitive.Popup>` (line 59)

Prop defaults: `side="bottom"`, `align="center"`, `sideOffset=4`, `alignOffset=0`, `tooltipStyle=false`

#### base

```txt
relative flex h-(--popup-height,auto) w-(--popup-width,auto) origin-(--transform-origin) rounded-lg border bg-popover not-dark:bg-clip-padding text-popover-foreground shadow-lg/5 outline-none transition-[width,height,scale,opacity] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] has-data-[slot=calendar]:rounded-xl has-data-[slot=calendar]:before:rounded-[calc(var(--radius-xl)-1px)] data-starting-style:scale-98 data-starting-style:opacity-0 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · height `auto` · width `auto` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · transition-property `width, height, scale, opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| height | `auto` | `var(--popup-height,auto)` | `h-(--popup-height,auto)` |
| width | `auto` | `var(--popup-width,auto)` | `w-(--popup-width,auto)` |
| transform-origin | `var(--transform-origin)` | `runtime var --transform-origin` | `origin-(--transform-origin)` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `width, height, scale, opacity` | `width,height,scale,opacity` | `transition-[width,height,scale,opacity]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[width,height,scale,opacity]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[width,height,scale,opacity]` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | border-radius | `14px` | `has-data-[slot=calendar]:rounded-xl` |
| starting-style (enter from) | scale | `98% 98%` | `data-starting-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | border-radius | `13px` | `has-data-[slot=calendar]:before:rounded-[calc(var(--radius-xl)-1px)]` |

#### when `tooltipStyle`

```txt
w-fit text-balance rounded-md text-xs shadow-md/5 before:rounded-[calc(var(--radius-md)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `fit-content` | `fit-content` | `w-fit` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| box-shadow | `0 4px 6px -1px #0000000d, 0 2px 4px -2px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-md/5` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| text-wrap | `balance` | `balance` | `text-balance` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | border-radius | `14px` | `has-data-[slot=calendar]:rounded-xl` |
| starting-style (enter from) | scale | `98% 98%` | `data-starting-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | border-radius | `13px` | `has-data-[slot=calendar]:before:rounded-[calc(var(--radius-xl)-1px)]` |

### `PopoverPopup` → `[data-slot=popover-viewport]` · `<PopoverPrimitive.Viewport>` (line 69)

Prop defaults: `side="bottom"`, `align="center"`, `sideOffset=4`, `alignOffset=0`, `tooltipStyle=false`

#### base

```txt
relative size-full max-h-(--available-height) overflow-clip px-(--viewport-inline-padding) py-4 [--viewport-inline-padding:--spacing(4)] has-data-[slot=calendar]:p-2 data-instant:transition-none **:data-current:data-ending-style:opacity-0 **:data-current:data-starting-style:opacity-0 **:data-previous:data-ending-style:opacity-0 **:data-previous:data-starting-style:opacity-0 **:data-current:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)] **:data-previous:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)] **:data-current:opacity-100 **:data-previous:opacity-100 **:data-current:transition-opacity **:data-previous:transition-opacity
```

**element — default**

Key metrics: height `100%` · max-height `var(--available-height)` (runtime var --available-height) · width `100%` · padding-inline `16px` (runtime var --viewport-inline-padding) · padding-block `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |
| max-height | `var(--available-height)` | `runtime var --available-height` | `max-h-(--available-height)` |
| overflow | `clip` | `clip` | `overflow-clip` |
| padding-inline | `16px` | `runtime var --viewport-inline-padding` | `px-(--viewport-inline-padding)` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | padding | `8px` | `has-data-[slot=calendar]:p-2` |
| [data-instant] | transition-property | `none` | `data-instant:transition-none` |

**`*[data-current]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `calc(var(--popup-width) - 2 * 16px - 2px)` | `calc(var(--popup-width) - 2 * var(--viewport-inline-padding) - 2px)` | `**:data-current:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)]` |
| opacity | `100%` | `100%` | `**:data-current:opacity-100` |
| transition-property | `opacity` | `opacity` | `**:data-current:transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `**:data-current:transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `**:data-current:transition-opacity` |

**`*[data-current][data-ending-style]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `0%` | `0%` | `**:data-current:data-ending-style:opacity-0` |

**`*[data-current][data-starting-style]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `0%` | `0%` | `**:data-current:data-starting-style:opacity-0` |

**`*[data-previous]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `calc(var(--popup-width) - 2 * 16px - 2px)` | `calc(var(--popup-width) - 2 * var(--viewport-inline-padding) - 2px)` | `**:data-previous:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)]` |
| opacity | `100%` | `100%` | `**:data-previous:opacity-100` |
| transition-property | `opacity` | `opacity` | `**:data-previous:transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `**:data-previous:transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `**:data-previous:transition-opacity` |

**`*[data-previous][data-ending-style]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `0%` | `0%` | `**:data-previous:data-ending-style:opacity-0` |

**`*[data-previous][data-starting-style]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `0%` | `0%` | `**:data-previous:data-starting-style:opacity-0` |

#### when `tooltipStyle`

```txt
py-1 [--viewport-inline-padding:--spacing(2)]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | padding | `8px` | `has-data-[slot=calendar]:p-2` |
| [data-instant] | transition-property | `none` | `data-instant:transition-none` |

#### when `!(tooltipStyle)`

```txt
not-data-transitioning:overflow-y-auto
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| overflow-y | `auto` | `auto` | `not-data-transitioning:overflow-y-auto` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=calendar]) | padding | `8px` | `has-data-[slot=calendar]:p-2` |
| [data-instant] | transition-property | `none` | `data-instant:transition-none` |
| [data-transitioning] | overflow-y | `(unset)` | `(removed)` |

### `PopoverClose` → `[data-slot=popover-close]` · `<PopoverPrimitive.Close>` (line 89)

### `PopoverTitle` → `[data-slot=popover-title]` · `<PopoverPrimitive.Title>` (line 97)

#### base

```txt
font-semibold text-lg leading-none
```

**element — default**

Key metrics: font-size `18px` (text-lg) · line-height `18px` · font-weight `600` (font-weight-semibold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `18px` | `text-lg` | `text-lg` |
| line-height | `18px` | `1` | `leading-none` |
| font-weight | `600` | `font-weight-semibold` | `font-semibold` |

### `PopoverDescription` → `[data-slot=popover-description]` · `<PopoverPrimitive.Description>` (line 110)

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

