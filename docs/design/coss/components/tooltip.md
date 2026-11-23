# Tooltip

Source: `apps/ui/registry/default/ui/tooltip.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/tooltip.mdx` · Base UI: https://base-ui.com/react/components/tooltip#api-reference

> A popup that appears when an element is hovered or focused, showing a hint for sighted users.

Depends on: `@base-ui/react/tooltip`

Exports: `TooltipTrigger`, `TooltipPopup`

## Behavior & keyboard

- Opens on hover after delay (Base UI default 600ms; `TooltipProvider` can share delay) and on keyboard focus; closes on `Escape`, blur, pointer leave. Not focusable itself.
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `TooltipTrigger` → `[data-slot=tooltip-trigger]` · `<TooltipPrimitive.Trigger>` (line 18)

### `TooltipPopup` → `[data-slot=tooltip-positioner]` · `<TooltipPrimitive.Positioner>` (line 39)

Attributes: `align={align}`, `anchor={anchor}`, `side={side}`, `sideOffset={sideOffset}`

Prop defaults: `align="center"`, `sideOffset=4`, `side="top"`

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

### `TooltipPopup` → `[data-slot=tooltip-popup]` · `<TooltipPrimitive.Popup>` (line 47)

Prop defaults: `align="center"`, `sideOffset=4`, `side="top"`

#### base

```txt
relative flex h-(--popup-height,auto) w-(--popup-width,auto) origin-(--transform-origin) text-balance rounded-md border bg-popover not-dark:bg-clip-padding text-popover-foreground text-xs shadow-md/5 transition-[width,height,scale,opacity] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-md)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] data-ending-style:scale-98 data-starting-style:scale-98 data-ending-style:opacity-0 data-starting-style:opacity-0 data-instant:duration-0 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · height `auto` · width `auto` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `8px` · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · font-size `12px` (text-xs) · line-height `16px` · box-shadow `0 4px 6px -1px #0000000d, 0 2px 4px -2px #0000000d` · transition-property `width, height, scale, opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

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
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| text-wrap | `balance` | `balance` | `text-balance` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 4px 6px -1px #0000000d, 0 2px 4px -2px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-md/5` |
| transition-property | `width, height, scale, opacity` | `width,height,scale,opacity` | `transition-[width,height,scale,opacity]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[width,height,scale,opacity]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[width,height,scale,opacity]` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | scale | `98% 98%` | `data-ending-style:scale-98` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `data-starting-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| [data-instant] | transition-duration | `0ms` | `data-instant:duration-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `TooltipPopup` → `[data-slot=tooltip-viewport]` · `<TooltipPrimitive.Viewport>` (line 55)

Prop defaults: `align="center"`, `sideOffset=4`, `side="top"`

#### base

```txt
relative size-full overflow-clip px-(--viewport-inline-padding) py-1 [--viewport-inline-padding:--spacing(2)] data-instant:transition-none **:data-current:data-ending-style:opacity-0 **:data-current:data-starting-style:opacity-0 **:data-previous:data-ending-style:opacity-0 **:data-previous:data-starting-style:opacity-0 **:data-current:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)] **:data-previous:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)] **:data-previous:truncate **:data-current:opacity-100 **:data-previous:opacity-100 **:data-current:transition-opacity **:data-previous:transition-opacity
```

**element — default**

Key metrics: height `100%` · width `100%` · padding-inline `8px` (runtime var --viewport-inline-padding) · padding-block `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |
| overflow | `clip` | `clip` | `overflow-clip` |
| padding-inline | `8px` | `runtime var --viewport-inline-padding` | `px-(--viewport-inline-padding)` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-instant] | transition-property | `none` | `data-instant:transition-none` |

**`*[data-current]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `calc(var(--popup-width) - 2 * 8px - 2px)` | `calc(var(--popup-width) - 2 * var(--viewport-inline-padding) - 2px)` | `**:data-current:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)]` |
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
| width | `calc(var(--popup-width) - 2 * 8px - 2px)` | `calc(var(--popup-width) - 2 * var(--viewport-inline-padding) - 2px)` | `**:data-previous:w-[calc(var(--popup-width)-2*var(--viewport-inline-padding)-2px)]` |
| overflow | `hidden` | `hidden` | `**:data-previous:truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `**:data-previous:truncate` |
| white-space | `nowrap` | `nowrap` | `**:data-previous:truncate` |
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

