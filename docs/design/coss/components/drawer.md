# Drawer

Source: `apps/ui/registry/default/ui/drawer.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/drawer.mdx` · Base UI: https://base-ui.com/react/components/drawer

> A panel that slides in from the edge of the screen with swipe gestures, snap points, and nested drawer support.

Depends on: `@base-ui/react/checkbox`, `@base-ui/react/drawer`, `@base-ui/react/merge-props`, `@base-ui/react/radio`, `@base-ui/react/radio-group`, `@base-ui/react/use-render`, `lucide-react`, `@/ui/button`, `@/ui/scroll-area`

Exports: `Drawer`, `DrawerTrigger`, `DrawerClose`, `DrawerSwipeArea`, `DrawerBackdrop`, `DrawerViewport`, `DrawerPopup`, `DrawerHeader`, `DrawerFooter`, `DrawerTitle`, `DrawerDescription`, `DrawerPanel`, `DrawerBar`, `DrawerMenu`, `DrawerMenuItem`, `DrawerMenuSeparator`, `DrawerMenuGroup`, `DrawerMenuGroupLabel`, `DrawerMenuTrigger`, `DrawerMenuCheckboxItem`, `DrawerMenuRadioGroup`, `DrawerMenuRadioItem`

## Behavior & keyboard

- Bottom/side sheet with swipe-to-dismiss (Base UI Drawer). Drag handle; `Escape` closes; focus trapped while open; swipe state exposed via `data-swiping`, `--drawer-swipe-*` CSS vars; nested drawers set `data-nested-drawer-open`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

Docs mentions:

- ### Software keyboard
- Bottom drawers automatically handle software-keyboard focus and spacing for inputs, textareas, and contenteditable fields. No additional prop is needed. Drawers without a focused input keep their normal layout. With snap points, the drawer moves up as needed while the keyboard is open and returns to the selected snap point when it closes.
- Put scrolling fields in `DrawerPanel`. For a persistent input or action row, place it after the panel, outside its scroll area. Set `--drawer-height` on `DrawerPopup` when the layout needs a fixed height; keyboard handling does not set the drawer height.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Drawer`

No own classes (re-export / wrapper of the primitive); prop defaults: `position="bottom"`.

### `DrawerTrigger` → `[data-slot=drawer-trigger]` · `<DrawerPrimitive.Trigger>` (line 59)

### `DrawerClose` → `[data-slot=drawer-close]` · `<DrawerPrimitive.Close>` (line 65)

### `DrawerSwipeArea` → `[data-slot=drawer-swipe-area]` · `<DrawerPrimitive.SwipeArea>` (line 79)

#### base

```txt
fixed z-50 touch-none
```

**element — default**

Key metrics: z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| z-index | `50` | `50` | `z-50` |
| touch-action | `none` | `none` | `touch-none` |

#### when `position === "bottom"`

```txt
inset-x-0 bottom-0 h-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `inset-x-0` |
| bottom | `0px` | `calc(var(--spacing) * 0)` | `bottom-0` |
| height | `32px` | `calc(var(--spacing) * 8)` | `h-8` |

#### when `position === "top"`

```txt
inset-x-0 top-0 h-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `inset-x-0` |
| top | `0px` | `calc(var(--spacing) * 0)` | `top-0` |
| height | `32px` | `calc(var(--spacing) * 8)` | `h-8` |

#### when `position === "left"`

```txt
inset-y-0 left-0 w-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `inset-y-0` |
| left | `0px` | `calc(var(--spacing) * 0)` | `left-0` |
| width | `32px` | `calc(var(--spacing) * 8)` | `w-8` |

#### when `position === "right"`

```txt
inset-y-0 right-0 w-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `inset-y-0` |
| right | `0px` | `calc(var(--spacing) * 0)` | `right-0` |
| width | `32px` | `calc(var(--spacing) * 8)` | `w-8` |

### `DrawerBackdrop` → `[data-slot=drawer-backdrop]` · `<DrawerPrimitive.Backdrop>` (line 99)

#### base

```txt
fixed inset-0 z-50 bg-black/32 opacity-[calc(1-var(--drawer-swipe-progress))] backdrop-blur-sm transition-opacity duration-450 ease-[cubic-bezier(0.32,0.72,0,1)] data-ending-style:opacity-0 data-starting-style:opacity-0 data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)] data-swiping:duration-0 supports-[-webkit-touch-callout:none]:absolute
```

**element — default**

Key metrics: background-color `#00000052` (color-black/32%) · opacity `calc(1 - var(--drawer-swipe-progress))` · z-index `50` · transition-property `opacity` · transition-duration `450ms` · transition-timing-function `cubic-bezier(0.32,0.72,0,1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| background-color | `#00000052` | `color-black/32%` | `bg-black/32` |
| opacity | `calc(1 - var(--drawer-swipe-progress))` | `calc(1 - var(--drawer-swipe-progress))` | `opacity-[calc(1-var(--drawer-swipe-progress))]` |
| -webkit-backdrop-filter | `blur(8px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-sm` |
| backdrop-filter | `blur(8px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-sm` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.32,0.72,0,1)` | `cubic-bezier(0.32,0.72,0,1)` | `ease-[cubic-bezier(0.32,0.72,0,1)]` |
| transition-duration | `450ms` | `450ms` | `duration-450` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| [data-swiping] | transition-duration | `0ms` | `data-swiping:duration-0` |

### `DrawerViewport` → `[data-slot=drawer-viewport]` · `<DrawerPrimitive.Viewport>` (line 120)

Prop defaults: `variant="default"`

#### base

```txt
fixed inset-0 z-50 [--bleed:--spacing(12)] [--inset:0px] touch-none
```

**element — default**

Key metrics: z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| touch-action | `none` | `none` | `touch-none` |

#### when `position === "bottom"`

```txt
grid grid-rows-[1fr_auto] pt-[max(--spacing(2),env(safe-area-inset-top,0px))] after:pointer-events-none after:fixed after:inset-x-0 after:bottom-0 after:h-(--bleed) after:bg-popover data-closed:after:opacity-0 has-[[data-swiping]]:after:opacity-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| grid-template-rows | `1fr auto` | `1fr auto` | `grid-rows-[1fr_auto]` |
| padding-top | `max(8px, env(safe-area-inset-top,0px))` | `max(calc(var(--spacing) * 2), env(safe-area-inset-top,0px))` | `pt-[max(--spacing(2),env(safe-area-inset-top,0px))]` |

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `after:bg-popover` |
| pointer-events | `none` | `none` | `after:pointer-events-none` |
| position | `fixed` | `fixed` | `after:fixed` |
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `after:inset-x-0` |
| bottom | `0px` | `calc(var(--spacing) * 0)` | `after:bottom-0` |
| height | `48px` | `runtime var --bleed` | `after:h-(--bleed)` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `after:bg-popover` |

**`::after` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-closed] | opacity | `0%` | `data-closed:after:opacity-0` |
| has([data-swiping]) | opacity | `0%` | `has-[[data-swiping]]:after:opacity-0` |

#### when `position === "top"`

```txt
grid grid-rows-[auto_1fr] pb-12
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| grid-template-rows | `auto 1fr` | `auto 1fr` | `grid-rows-[auto_1fr]` |
| padding-bottom | `48px` | `calc(var(--spacing) * 12)` | `pb-12` |

#### when `position === "left"`

```txt
flex justify-start
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| justify-content | `flex-start` | `flex-start` | `justify-start` |

#### when `position === "right"`

```txt
flex justify-end
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| justify-content | `flex-end` | `flex-end` | `justify-end` |

#### when `variant === "inset"`

```txt
px-(--inset) sm:[--inset:--spacing(4)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `16px` | `runtime var --inset` | `px-(--inset)` |

**element — mobile differences**: padding-inline `0px`

#### when `variant === "inset" && position !== "bottom"`

```txt
pt-(--inset)
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-top | `0px` | `runtime var --inset` | `pt-(--inset)` |

#### when `variant === "inset" && position !== "top"`

```txt
pb-(--inset)
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-bottom | `0px` | `runtime var --inset` | `pb-(--inset)` |

#### when `variant === "inset" && position === "bottom"`

```txt
sm:after:hidden
```

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `sm:after:hidden` |
| display | `none` | `none` | `sm:after:hidden` |

**`::after` — mobile differences**: content `(unset)` · display `(unset)`

### `DrawerPopup` → `[data-slot=drawer-popup]` · `<DrawerPrimitive.Popup>` (line 164)

Prop defaults: `showCloseButton=false`, `variant="default"`, `showBar=false`

#### base

```txt
relative flex max-h-full min-h-0 w-full min-w-0 flex-col bg-popover not-dark:bg-clip-padding text-popover-foreground shadow-lg/5 outline-none transition-[transform,box-shadow,height,background-color] duration-450 ease-[cubic-bezier(0.32,0.72,0,1)] will-change-transform [--peek:calc(--spacing(6)-1px)] [--scale-base:calc(max(0,1-(var(--nested-drawers)*var(--stack-step))))] [--scale:clamp(0,calc(var(--scale-base)+(var(--stack-step)*var(--stack-progress))),1)] [--shrink:calc(1-var(--scale))] [--stack-peek-offset:max(0px,calc((var(--nested-drawers)-var(--stack-progress))*var(--peek)))] [--stack-progress:clamp(0,var(--drawer-swipe-progress),1)] [--stack-step:0.05] before:pointer-events-none before:absolute before:inset-0 before:shadow-[0_1px_--theme(--color-black/4%)] after:pointer-events-none after:absolute after:bg-popover data-swiping:select-none data-nested-drawer-open:overflow-hidden data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))] data-ending-style:shadow-transparent data-starting-style:shadow-transparent data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)] dark:data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(6%*(var(--nested-drawers)-var(--stack-progress))))] dark:before:shadow-[0_-1px_--theme(--color-white/6%)] touch-none
```

**element — default**

Key metrics: display `flex` · min-height `0px` · max-height `100%` · width `100%` · min-width `0px` · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · transition-property `transform, box-shadow, height, background-color` · transition-duration `450ms` · transition-timing-function `cubic-bezier(0.32,0.72,0,1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| max-height | `100%` | `100%` | `max-h-full` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| touch-action | `none` | `none` | `touch-none` |
| flex-direction | `column` | `column` | `flex-col` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `transform, box-shadow, height, background-color` | `transform,box-shadow,height,background-color` | `transition-[transform,box-shadow,height,background-color]` |
| transition-timing-function | `cubic-bezier(0.32,0.72,0,1)` | `cubic-bezier(0.32,0.72,0,1)` | `ease-[cubic-bezier(0.32,0.72,0,1)]` |
| transition-duration | `450ms` | `450ms` | `duration-450` |
| will-change | `transform` | `transform` | `will-change-transform` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `after:bg-popover` |
| pointer-events | `none` | `none` | `after:pointer-events-none` |
| position | `absolute` | `absolute` | `after:absolute` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `after:bg-popover` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

#### when `position === "bottom"`

```txt
z-1 transform-[translateY(calc(var(--keyboard-snap-offset)+var(--drawer-swipe-movement-y)))] data-ending-style:transform-[translateY(calc(100%+env(safe-area-inset-bottom,0px)+var(--inset)))] data-starting-style:transform-[translateY(calc(100%+env(safe-area-inset-bottom,0px)+var(--inset)))] row-start-2 -mb-[max(0px,calc(var(--keyboard-snap-offset)+clamp(0,1,var(--keyboard-snap-offset)/1px)*var(--drawer-swipe-movement-y,0px)))] border-t pb-[max(0px,calc(max(env(safe-area-inset-bottom,0px),var(--drawer-keyboard-inset,0px))+var(--keyboard-snap-offset)+clamp(0,1,var(--keyboard-snap-offset)/1px)*var(--drawer-swipe-movement-y,0px)))] not-data-starting-style:not-data-ending-style:transition-[transform,box-shadow,height,background-color,margin,padding] [--keyboard-snap-offset:max(0px,calc(var(--drawer-snap-point-offset,0px)-var(--drawer-keyboard-inset,0px)))] after:inset-x-0 after:top-full after:h-(--bleed) has-data-[slot=drawer-bar]:pt-2 data-ending-style:mb-0 data-starting-style:mb-0 data-ending-style:pb-0 data-starting-style:pb-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transition-property | `transform, box-shadow, height, background-color, margin, padding` | `transform,box-shadow,height,background-color,margin,padding` | `not-data-starting-style:not-data-ending-style:transition-[transform,box-shadow,height,background-color,margin,padding]` |
| z-index | `1` | `1` | `z-1` |
| grid-row-start | `2` | `2` | `row-start-2` |
| margin-bottom | `calc(max(0px, calc(max(0px, 0px) + clamp(0, 1, max(0px, 0px) / 1px) * 0px)) * -1)` | `calc(max(0px, calc(var(--keyboard-snap-offset) + clamp(0, 1, var(--keyboard-snap-offset) / 1px) * var(--drawer` | `-mb-[max(0px,calc(var(--keyboard-snap-offset)+clamp(0,1,var(--keyboard-snap-offset)/1px)*var(--drawer-swipe-movement-y,0px)))]` |
| transform | `translateY(calc(max(0px, 0px) + var(--drawer-swipe-movement-y)))` | `translateY(calc(var(--keyboard-snap-offset) + var(--drawer-swipe-movement-y)))` | `transform-[translateY(calc(var(--keyboard-snap-offset)+var(--drawer-swipe-movement-y)))]` |
| border-top-style | `solid` | `var(--tw-border-style)` | `border-t` |
| border-top-width | `1px` | `1px` | `border-t` |
| padding-bottom | `max(0px, calc(max(env(safe-area-inset-bottom,0px), 0px) + max(0px, 0px) + clamp(0, 1, max(0px, 0px) / 1px) * 0px))` | `max(0px, calc(max(env(safe-area-inset-bottom,0px), var(--drawer-keyboard-inset,0px)) + var(--keyboard-snap-off` | `pb-[max(0px,calc(max(env(safe-area-inset-bottom,0px),var(--drawer-keyboard-inset,0px))+var(--keyboard-snap-offset)+clamp(0,1,var(--keyboard-snap-offset)/1px)*var(--drawer-swipe-movement-y,0px)))]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | margin-bottom | `0px` | `data-ending-style:mb-0` |
| ending-style (exit to) | transform | `translateY(calc(100% + env(safe-area-inset-bottom,0px) + 0px))` | `data-ending-style:transform-[translateY(calc(100%+env(safe-area-inset-bottom,0px)+var(--inset)))]` |
| ending-style (exit to) | padding-bottom | `0px` | `data-ending-style:pb-0` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-property | `transform, box-shadow, height, background-color` | `transition-[transform,box-shadow,height,background-color]` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | margin-bottom | `0px` | `data-starting-style:mb-0` |
| starting-style (enter from) | transform | `translateY(calc(100% + env(safe-area-inset-bottom,0px) + 0px))` | `data-starting-style:transform-[translateY(calc(100%+env(safe-area-inset-bottom,0px)+var(--inset)))]` |
| starting-style (enter from) | padding-bottom | `0px` | `data-starting-style:pb-0` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |
| starting-style (enter from) | transition-property | `transform, box-shadow, height, background-color` | `transition-[transform,box-shadow,height,background-color]` |
| has([data-slot=drawer-bar]) | padding-top | `8px` | `has-data-[slot=drawer-bar]:pt-2` |

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `after:inset-x-0` |
| top | `100%` | `100%` | `after:top-full` |
| height | `48px` | `runtime var --bleed` | `after:h-(--bleed)` |

#### when `position === "top"`

```txt
data-starting-style:transform-[translateY(calc(-100%-var(--inset)))] data-ending-style:transform-[translateY(calc(-100%-var(--inset)))] transform-[translateY(var(--drawer-swipe-movement-y))] border-b after:inset-x-0 after:bottom-full after:h-(--bleed) has-data-[slot=drawer-bar]:pb-2
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform | `translateY(var(--drawer-swipe-movement-y))` | `translateY(var(--drawer-swipe-movement-y))` | `transform-[translateY(var(--drawer-swipe-movement-y))]` |
| border-bottom-style | `solid` | `var(--tw-border-style)` | `border-b` |
| border-bottom-width | `1px` | `1px` | `border-b` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | transform | `translateY(calc(-100% - 0px))` | `data-ending-style:transform-[translateY(calc(-100%-var(--inset)))]` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | transform | `translateY(calc(-100% - 0px))` | `data-starting-style:transform-[translateY(calc(-100%-var(--inset)))]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |
| has([data-slot=drawer-bar]) | padding-bottom | `8px` | `has-data-[slot=drawer-bar]:pb-2` |

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `after:inset-x-0` |
| bottom | `100%` | `100%` | `after:bottom-full` |
| height | `48px` | `runtime var --bleed` | `after:h-(--bleed)` |

#### when `position === "left"`

```txt
data-starting-style:transform-[translateX(calc(-100%-var(--inset)))] data-ending-style:transform-[translateX(calc(-100%-var(--inset)))] transform-[translateX(var(--drawer-swipe-movement-x))] w-[calc(100%-(--spacing(12)))] max-w-md border-e after:inset-y-0 after:end-full after:w-(--bleed) has-data-[slot=drawer-bar]:pe-2
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| max-width | `448px` | `container-md` | `max-w-md` |
| transform | `translateX(var(--drawer-swipe-movement-x))` | `translateX(var(--drawer-swipe-movement-x))` | `transform-[translateX(var(--drawer-swipe-movement-x))]` |
| border-inline-end-style | `solid` | `var(--tw-border-style)` | `border-e` |
| border-inline-end-width | `1px` | `1px` | `border-e` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | transform | `translateX(calc(-100% - 0px))` | `data-ending-style:transform-[translateX(calc(-100%-var(--inset)))]` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | transform | `translateX(calc(-100% - 0px))` | `data-starting-style:transform-[translateX(calc(-100%-var(--inset)))]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |
| has([data-slot=drawer-bar]) | padding-inline-end | `8px` | `has-data-[slot=drawer-bar]:pe-2` |

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `after:inset-y-0` |
| inset-inline-end | `100%` | `100%` | `after:end-full` |
| width | `48px` | `runtime var --bleed` | `after:w-(--bleed)` |

#### when `position === "right"`

```txt
transform-[translateX(var(--drawer-swipe-movement-x))] data-ending-style:transform-[translateX(calc(100%+var(--inset)))] data-starting-style:transform-[translateX(calc(100%+var(--inset)))] col-start-2 w-[calc(100%-(--spacing(12)))] max-w-md border-s after:inset-y-0 after:start-full after:w-(--bleed) has-data-[slot=drawer-bar]:ps-2
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| max-width | `448px` | `container-md` | `max-w-md` |
| transform | `translateX(var(--drawer-swipe-movement-x))` | `translateX(var(--drawer-swipe-movement-x))` | `transform-[translateX(var(--drawer-swipe-movement-x))]` |
| border-inline-start-style | `solid` | `var(--tw-border-style)` | `border-s` |
| border-inline-start-width | `1px` | `1px` | `border-s` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | transform | `translateX(calc(100% + 0px))` | `data-ending-style:transform-[translateX(calc(100%+var(--inset)))]` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | transform | `translateX(calc(100% + 0px))` | `data-starting-style:transform-[translateX(calc(100%+var(--inset)))]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |
| has([data-slot=drawer-bar]) | padding-inline-start | `8px` | `has-data-[slot=drawer-bar]:ps-2` |

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `after:inset-y-0` |
| inset-inline-start | `100%` | `100%` | `after:start-full` |
| width | `48px` | `runtime var --bleed` | `after:w-(--bleed)` |

#### when `variant !== "straight" && position === "bottom"`

```txt
rounded-t-2xl
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-top-left-radius | `16px` | `radius-2xl` | `rounded-t-2xl` |
| border-top-right-radius | `16px` | `radius-2xl` | `rounded-t-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

#### when `variant !== "straight" && position === "top"`

```txt
rounded-b-2xl **:data-[slot=drawer-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `16px` | `radius-2xl` | `rounded-b-2xl` |
| border-bottom-left-radius | `16px` | `radius-2xl` | `rounded-b-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`*[data-slot="drawer-footer"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `**:data-[slot=drawer-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-bottom-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `**:data-[slot=drawer-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]` |

#### when `variant !== "straight" && position === "left"`

```txt
rounded-e-2xl **:data-[slot=drawer-footer]:rounded-ee-[calc(var(--radius-2xl)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `16px` | `radius-2xl` | `rounded-e-2xl` |
| border-end-end-radius | `16px` | `radius-2xl` | `rounded-e-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`*[data-slot="drawer-footer"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-end-end-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `**:data-[slot=drawer-footer]:rounded-ee-[calc(var(--radius-2xl)-1px)]` |

#### when `variant !== "straight" && position === "right"`

```txt
rounded-s-2xl **:data-[slot=drawer-footer]:rounded-es-[calc(var(--radius-2xl)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `16px` | `radius-2xl` | `rounded-s-2xl` |
| border-end-start-radius | `16px` | `radius-2xl` | `rounded-s-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`*[data-slot="drawer-footer"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-end-start-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `**:data-[slot=drawer-footer]:rounded-es-[calc(var(--radius-2xl)-1px)]` |

#### when `variant === "default" && position === "bottom"`

```txt
before:rounded-t-[calc(var(--radius-2xl)-1px)]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-top-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-t-[calc(var(--radius-2xl)-1px)]` |
| border-top-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-t-[calc(var(--radius-2xl)-1px)]` |

#### when `variant === "default" && position === "top"`

```txt
before:rounded-b-[calc(var(--radius-2xl)-1px)]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-bottom-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-b-[calc(var(--radius-2xl)-1px)]` |

#### when `variant === "default" && position === "left"`

```txt
before:rounded-e-[calc(var(--radius-2xl)-1px)]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-e-[calc(var(--radius-2xl)-1px)]` |
| border-end-end-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-e-[calc(var(--radius-2xl)-1px)]` |

#### when `variant === "default" && position === "right"`

```txt
before:rounded-s-[calc(var(--radius-2xl)-1px)]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-s-[calc(var(--radius-2xl)-1px)]` |
| border-end-start-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-s-[calc(var(--radius-2xl)-1px)]` |

#### when `variant === "inset"`

```txt
before:hidden sm:rounded-2xl sm:border sm:after:bg-transparent sm:before:rounded-[calc(var(--radius-2xl)-1px)] sm:**:data-[slot=drawer-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `sm:border` |
| border-style | `solid` | `var(--tw-border-style)` | `sm:border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| border-radius | `16px` | `radius-2xl` | `sm:rounded-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

**element — mobile differences**: border-width `(unset)` · border-style `(unset)` · border-color `(unset)` · border-radius `(unset)`

**`::after` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#00000000` | `transparent` | `sm:after:bg-transparent` |

**`::after` — mobile differences**: background-color `#ffffff` / `#1d1d1d`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `none` | `none` | `before:hidden` |
| border-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:before:rounded-[calc(var(--radius-2xl)-1px)]` |

**`::before` — mobile differences**: border-radius `(unset)`

**`*[data-slot="drawer-footer"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:**:data-[slot=drawer-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-bottom-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:**:data-[slot=drawer-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]` |

**`*[data-slot="drawer-footer"]` — mobile differences**: border-bottom-right-radius `(unset)` · border-bottom-left-radius `(unset)`

#### when `variant === "straight"`

```txt
[--stack-step:0]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

#### when `(position === "bottom" || position === "top")`

```txt
h-(--drawer-height,auto) [--height:max(0px,calc(var(--drawer-frontmost-height,var(--drawer-height))))] data-nested-drawer-open:h-(--height)
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `auto` | `var(--drawer-height,auto)` | `h-(--drawer-height,auto)` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | height | `max(0px, calc(var(--drawer-height)))` | `data-nested-drawer-open:h-(--height)` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

#### when `position === "bottom"`

```txt
data-nested-drawer-open:transform-[translateY(calc(var(--drawer-swipe-movement-y)-var(--stack-peek-offset)-(var(--shrink)*var(--height))))_scale(var(--scale))] origin-[50%_calc(100%-var(--inset))]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `50% calc(100% - 0px)` | `50% calc(100% - var(--inset))` | `origin-[50%_calc(100%-var(--inset))]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | transform | `translateY(calc(var(--drawer-swipe-movement-y) - max(0px, calc((var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1)) * 23px)) - (calc(1 - clamp(0, calc(calc(max(0, 1 - (var(--nested-drawers) * 0.05))) + (0.05 * clamp(0, var(--drawer-swipe-progress), 1))), 1)) * var(--height)))) scale(clamp(0, calc(calc(max(0, 1 - (var(--nested-drawers) * 0.05))) + (0.05 * clamp(0, var(--drawer-swipe-progress), 1))), 1))` | `data-nested-drawer-open:transform-[translateY(calc(var(--drawer-swipe-movement-y)-var(--stack-peek-offset)-(var(--shrink)*var(--height))))_scale(var(--scale))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

#### when `position === "top"`

```txt
data-nested-drawer-open:transform-[translateY(calc(var(--drawer-swipe-movement-y)+var(--stack-peek-offset)+(var(--shrink)*var(--height))))_scale(var(--scale))] origin-[50%_var(--inset)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `50% 0px` | `50% var(--inset)` | `origin-[50%_var(--inset)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | transform | `translateY(calc(var(--drawer-swipe-movement-y) + max(0px, calc((var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1)) * 23px)) + (calc(1 - clamp(0, calc(calc(max(0, 1 - (var(--nested-drawers) * 0.05))) + (0.05 * clamp(0, var(--drawer-swipe-progress), 1))), 1)) * var(--height)))) scale(clamp(0, calc(calc(max(0, 1 - (var(--nested-drawers) * 0.05))) + (0.05 * clamp(0, var(--drawer-swipe-progress), 1))), 1))` | `data-nested-drawer-open:transform-[translateY(calc(var(--drawer-swipe-movement-y)+var(--stack-peek-offset)+(var(--shrink)*var(--height))))_scale(var(--scale))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

#### when `position === "left"`

```txt
data-nested-drawer-open:transform-[translateX(calc(var(--drawer-swipe-movement-x)+var(--stack-peek-offset)))_scale(var(--scale))] origin-right
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `100%` | `100%` | `origin-right` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | transform | `translateX(calc(var(--drawer-swipe-movement-x) + max(0px, calc((var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1)) * 23px)))) scale(clamp(0, calc(calc(max(0, 1 - (var(--nested-drawers) * 0.05))) + (0.05 * clamp(0, var(--drawer-swipe-progress), 1))), 1))` | `data-nested-drawer-open:transform-[translateX(calc(var(--drawer-swipe-movement-x)+var(--stack-peek-offset)))_scale(var(--scale))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

#### when `position === "right"`

```txt
data-nested-drawer-open:transform-[translateX(calc(var(--drawer-swipe-movement-x)-var(--stack-peek-offset)))_scale(var(--scale))] origin-left
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `0` | `0` | `origin-left` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-swiping] | -webkit-user-select | `none` | `data-swiping:select-none` |
| [data-swiping] | user-select | `none` | `data-swiping:select-none` |
| [data-nested-drawer-open] | background-color | `color-mix(in srgb,#fff,#000 calc(2% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * (var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1))))` | `data-nested-drawer-open:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(2%*(var(--nested-drawers)-var(--stack-progress))))]` |
| [data-nested-drawer-open] | transform | `translateX(calc(var(--drawer-swipe-movement-x) - max(0px, calc((var(--nested-drawers) - clamp(0, var(--drawer-swipe-progress), 1)) * 23px)))) scale(clamp(0, calc(calc(max(0, 1 - (var(--nested-drawers) * 0.05))) + (0.05 * clamp(0, var(--drawer-swipe-progress), 1))), 1))` | `data-nested-drawer-open:transform-[translateX(calc(var(--drawer-swipe-movement-x)-var(--stack-peek-offset)))_scale(var(--scale))]` |
| [data-nested-drawer-open] | overflow | `hidden` | `data-nested-drawer-open:overflow-hidden` |
| ending-style (exit to) | box-shadow | `none` | `shadow-lg/5` |
| ending-style (exit to) | transition-duration | `calc(var(--drawer-swipe-strength) * 400ms)` | `data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]` |
| starting-style (enter from) | box-shadow | `none` | `shadow-lg/5` |

### `DrawerPopup` → (no data-slot) · `<DrawerPrimitive.Close>` (line 217)

Attributes: `aria-label=Close`, `render={<Button size="icon" variant="ghost" />}`

Prop defaults: `showCloseButton=false`, `variant="default"`, `showBar=false`

#### base

```txt
absolute end-2 top-2 z-1
```

**element — default**

Key metrics: z-index `1`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset-inline-end | `8px` | `calc(var(--spacing) * 2)` | `end-2` |
| top | `8px` | `calc(var(--spacing) * 2)` | `top-2` |
| z-index | `1` | `1` | `z-1` |

### `DrawerHeader` → `[data-slot=drawer-header]` · `useRender <div>` (line 248)

Prop defaults: `allowSelection=false`

#### base

```txt
flex flex-col gap-2 p-6 in-[[data-slot=drawer-popup]:has([data-slot=drawer-panel])]:pb-3 max-sm:pb-4
```

**element — default**

Key metrics: display `flex` · padding `24px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-panel])))) | padding-bottom | `12px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-panel])]:pb-3` |

**element — mobile differences**: padding-bottom `16px`

#### when `!allowSelection`

```txt
cursor-default
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| cursor | `default` | `default` | `cursor-default` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-panel])))) | padding-bottom | `12px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-panel])]:pb-3` |

**element — mobile differences**: padding-bottom `16px`

### `DrawerFooter` → `[data-slot=drawer-footer]` · `useRender <div>` (line 274)

Prop defaults: `variant="default"`, `allowSelection=true`

#### base

```txt
flex flex-col-reverse gap-2 px-6 pb-(--safe-area-inset-bottom,0px) sm:flex-row sm:justify-end
```

**element — default**

Key metrics: display `flex` · padding-inline `24px` · padding-bottom `0px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `row` | `row` | `sm:flex-row` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding-inline | `24px` | `calc(var(--spacing) * 6)` | `px-6` |
| padding-bottom | `0px` | `var(--safe-area-inset-bottom,0px)` | `pb-(--safe-area-inset-bottom,0px)` |
| justify-content | `flex-end` | `flex-end` | `sm:justify-end` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

#### when `!allowSelection`

```txt
cursor-default
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| cursor | `default` | `default` | `cursor-default` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

#### when `variant === "default"`

```txt
border-t bg-muted/72 pt-4 pb-[calc(env(safe-area-inset-bottom,0px)+--spacing(4))]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-bottom | `calc(env(safe-area-inset-bottom,0px) + 16px)` | `calc(env(safe-area-inset-bottom,0px) + calc(var(--spacing) * 4))` | `pb-[calc(env(safe-area-inset-bottom,0px)+--spacing(4))]` |
| border-top-style | `solid` | `var(--tw-border-style)` | `border-t` |
| border-top-width | `1px` | `1px` | `border-t` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `bg-muted/72` |
| padding-top | `16px` | `calc(var(--spacing) * 4)` | `pt-4` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

#### when `variant === "bare"`

```txt
in-[[data-slot=drawer-popup]:has([data-slot=drawer-panel])]:pt-3 pt-4 pb-[calc(env(safe-area-inset-bottom,0px)+--spacing(6))]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-bottom | `calc(env(safe-area-inset-bottom,0px) + 24px)` | `calc(env(safe-area-inset-bottom,0px) + calc(var(--spacing) * 6))` | `pb-[calc(env(safe-area-inset-bottom,0px)+--spacing(6))]` |
| padding-top | `16px` | `calc(var(--spacing) * 4)` | `pt-4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-panel])))) | padding-top | `12px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-panel])]:pt-3` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

### `DrawerTitle` → `[data-slot=drawer-title]` · `<DrawerPrimitive.Title>` (line 299)

#### base

```txt
font-bold font-heading text-xl leading-none
```

**element — default**

Key metrics: font-size `20px` (text-xl) · line-height `20px` · font-weight `700` (font-weight-bold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-family | `ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` | `font-sans` | `font-heading` |
| font-size | `20px` | `text-xl` | `text-xl` |
| line-height | `20px` | `1` | `leading-none` |
| font-weight | `700` | `font-weight-bold` | `font-bold` |

### `DrawerDescription` → `[data-slot=drawer-description]` · `<DrawerPrimitive.Description>` (line 312)

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

### `DrawerPanel` → `[data-slot=drawer-panel]` · `useRender <div>` (line 332)

Prop defaults: `scrollFade=true`, `scrollable=true`, `allowSelection=true`

#### base

```txt
p-6 in-[[data-slot=drawer-popup]:has([data-slot=drawer-header])]:pt-1 in-[[data-slot=drawer-popup]:has([data-slot=drawer-footer]:not(.border-t))]:pb-1
```

**element — default**

Key metrics: padding `24px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-header])))) | padding-top | `4px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-header])]:pt-1` |
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-footer]:not(.border-t))))) | padding-bottom | `4px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-footer]:not(.border-t))]:pb-1` |

#### when `!allowSelection`

```txt
cursor-default
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| cursor | `default` | `default` | `cursor-default` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-header])))) | padding-top | `4px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-header])]:pt-1` |
| in(:where(*:is([data-slot=drawer-popup]:has([data-slot=drawer-footer]:not(.border-t))))) | padding-bottom | `4px` | `in-[[data-slot=drawer-popup]:has([data-slot=drawer-footer]:not(.border-t))]:pb-1` |

### `DrawerPanel` → (no data-slot) · `<ScrollArea>` (line 350)

Attributes: `overscrollContain=true`, `scrollFade={scrollFade}`

Prop defaults: `scrollFade=true`, `scrollable=true`, `allowSelection=true`

#### base

```txt
touch-auto *:data-[slot=scroll-area-viewport]:pb-0!
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| touch-action | `auto` | `auto` | `touch-auto` |

**`> *[data-slot="scroll-area-viewport"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-bottom | `0px` | `calc(var(--spacing) * 0)` | `*:data-[slot=scroll-area-viewport]:pb-0!` |

### `DrawerBar` → `[data-slot=drawer-bar]` · `useRender <div>` (line 374)

Attributes: `aria-hidden={true as const}`

#### base

```txt
absolute flex touch-none items-center justify-center p-3 before:rounded-full before:bg-input
```

**element — default**

Key metrics: display `flex` · padding `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| display | `flex` | `flex` | `flex` |
| touch-action | `none` | `none` | `touch-none` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| padding | `12px` | `calc(var(--spacing) * 3)` | `p-3` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:bg-input` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `before:bg-input` |

#### when `horizontal`

```txt
inset-y-0 before:h-12 before:w-1
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `inset-y-0` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `48px` | `calc(var(--spacing) * 12)` | `before:h-12` |
| width | `4px` | `calc(var(--spacing) * 1)` | `before:w-1` |

#### when `!(horizontal)`

```txt
inset-x-0 before:h-1 before:w-12
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `inset-x-0` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `4px` | `calc(var(--spacing) * 1)` | `before:h-1` |
| width | `48px` | `calc(var(--spacing) * 12)` | `before:w-12` |

#### when `position === "top"`

```txt
bottom-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| bottom | `0px` | `calc(var(--spacing) * 0)` | `bottom-0` |

#### when `position === "bottom"`

```txt
top-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| top | `0px` | `calc(var(--spacing) * 0)` | `top-0` |

#### when `position === "left"`

```txt
right-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| right | `0px` | `calc(var(--spacing) * 0)` | `right-0` |

#### when `position === "right"`

```txt
left-0
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| left | `0px` | `calc(var(--spacing) * 0)` | `left-0` |

### `DrawerMenu` → `[data-slot=drawer-menu]` · `useRender <nav>` (line 405)

#### base

```txt
-m-2 flex flex-col
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `-8px` | `calc(var(--spacing) * -2)` | `-m-2` |
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |

### `DrawerMenuItem` → `[data-slot=drawer-menu-item]` · `useRender <button>` (line 426)

Attributes: `data-variant={variant}`, `type={"button" as const}`

Prop defaults: `variant="default"`

#### base

```txt
flex min-h-9 w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-64 data-[variant=destructive]:text-destructive-foreground sm:min-h-8 sm:text-sm [&>svg:not([class*='opacity-'])]:opacity-80 [&>svg:not([class*='size-'])]:size-4.5 sm:[&>svg:not([class*='size-'])]:size-4 [&>svg]:pointer-events-none [&>svg]:-mx-0.5 [&>svg]:shrink-0
```

**element — default**

Key metrics: display `flex` · min-height `32px` · width `100%` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-variant=destructive] | color | `#c10007` / `#ff6467` | `data-[variant=destructive]:text-destructive-foreground` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&>svg]:pointer-events-none` |
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&>svg]:-mx-0.5` |
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

**`>svg:not([class*='opacity-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `[&>svg:not([class*='opacity-'])]:opacity-80` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&>svg:not([class*='size-'])]:size-4` |

**`>svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `DrawerMenuSeparator` → `[data-slot=drawer-menu-separator]` · `useRender <div>` (line 449)

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

### `DrawerMenuGroup` → `[data-slot=drawer-menu-group]` · `useRender <div>` (line 466)

#### base

```txt
flex flex-col
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |

### `DrawerMenuGroupLabel` → `[data-slot=drawer-menu-group-label]` · `useRender <div>` (line 483)

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

### `DrawerMenuTrigger` → `[data-slot=drawer-menu-trigger]` · `<DrawerTrigger>` (line 504)

#### base

```txt
flex min-h-9 w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none hover:bg-accent hover:text-accent-foreground sm:min-h-8 sm:text-sm [&_svg:not(:last-child)]:-mx-0.5 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `flex` · min-height `32px` · width `100%` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not(:last-child)` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg:not(:last-child)]:-mx-0.5` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `DrawerMenuTrigger` → (no data-slot) · `<ChevronRightIcon>` (line 513)

#### base

```txt
ms-auto -me-0.5 opacity-80
```

**element — default**

Key metrics: opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-start | `auto` | `auto` | `ms-auto` |
| margin-inline-end | `-2px` | `calc(var(--spacing) * -0.5)` | `-me-0.5` |
| opacity | `80%` | `80%` | `opacity-80` |

### `DrawerMenuCheckboxItem` → `[data-slot=drawer-menu-checkbox-item]` · `<CheckboxPrimitive.Root>` (line 533)

Attributes: `checked={checked}`, `defaultChecked={defaultChecked}`, `disabled={disabled}`, `render={render}`

Prop defaults: `variant="default"`

#### base

```txt
grid min-h-9 w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none hover:bg-accent hover:text-accent-foreground data-disabled:pointer-events-none data-disabled:opacity-64 sm:min-h-8 sm:text-sm [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0
```

**element — default**

Key metrics: display `grid` · min-height `32px` · width `100%` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-mx-0.5` |
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

#### when `variant === "switch"`

```txt
grid-cols-[minmax(0,1fr)_auto] gap-4 pe-1.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| gap | `16px` | `calc(var(--spacing) * 4)` | `gap-4` |
| grid-template-columns | `minmax(0,1fr) auto` | `minmax(0,1fr) auto` | `grid-cols-[minmax(0,1fr)_auto]` |
| padding-inline-end | `6px` | `calc(var(--spacing) * 1.5)` | `pe-1.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### when `!(variant === "switch")`

```txt
grid-cols-[1rem_minmax(0,1fr)] pe-4
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-template-columns | `16px minmax(0,1fr)` | `1rem minmax(0,1fr)` | `grid-cols-[1rem_minmax(0,1fr)]` |
| padding-inline-end | `16px` | `calc(var(--spacing) * 4)` | `pe-4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `DrawerMenuCheckboxItem` → (no data-slot) · `<span>` (line 551)

Prop defaults: `variant="default"`

#### base

```txt
wrap-anywhere col-start-1 min-w-0
```

**element — default**

Key metrics: min-width `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| overflow-wrap | `anywhere` | `anywhere` | `wrap-anywhere` |

### `DrawerMenuCheckboxItem` → (no data-slot) · `<CheckboxPrimitive.Indicator>` (line 552)

Attributes: `keepMounted=true`

Prop defaults: `variant="default"`

#### base

```txt
inset-shadow-[0_1px_--theme(--color-black/4%)] col-start-2 inline-flex h-[calc(var(--thumb-size)+2px)] w-[calc(var(--thumb-size)*2-2px)] shrink-0 items-center rounded-full p-px outline-none transition-[background-color,box-shadow] duration-200 [--thumb-size:--spacing(4)] focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background data-checked:bg-primary data-unchecked:bg-input data-disabled:opacity-64 sm:[--thumb-size:--spacing(3)]
```

**element — default**

Key metrics: display `inline-flex` · height `14px` · width `22px` · padding `1px` · border-radius `9999px` · box-shadow `inset 0 1px #0000000a` · transition-property `background-color, box-shadow` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `14px` | `calc(var(--thumb-size) + 2px)` | `h-[calc(var(--thumb-size)+2px)]` |
| width | `22px` | `calc(var(--thumb-size) * 2 - 2px)` | `w-[calc(var(--thumb-size)*2-2px)]` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| padding | `1px` | `1px` | `p-px` |
| box-shadow | `inset 0 1px #0000000a` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `inset-shadow-[0_1px_--theme(--color-black/4%)]` |
| transition-property | `background-color, box-shadow` | `background-color,box-shadow` | `transition-[background-color,box-shadow]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[background-color,box-shadow]` |
| transition-duration | `200ms` | `200ms` | `duration-200` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #0000000a, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `inset 0 1px #0000000a, 0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| checked | background-color | `#262626` / `#f5f5f5` | `data-checked:bg-primary` |
| unchecked | background-color | `#0000001a` / `#ffffff14` | `data-unchecked:bg-input` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: height `18px` · width `30px`

### `DrawerMenuCheckboxItem` → (no data-slot) · `<span>` (line 556)

Prop defaults: `variant="default"`

#### base

```txt
pointer-events-none block aspect-square h-full in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:origin-[var(--thumb-size)_50%] origin-left in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:translate-x-[calc(var(--thumb-size)-4px)] in-[[data-slot=drawer-menu-checkbox-item]:active]:not-data-disabled:scale-x-110 in-[[data-slot=drawer-menu-checkbox-item]:active]:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.10)] rounded-(--thumb-size) bg-background shadow-sm/5 will-change-transform [transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s] rtl:in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:origin-left rtl:origin-right rtl:in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:-translate-x-[calc(var(--thumb-size)-4px)]
```

**element — default**

Key metrics: display `block` · height `100%` · border-radius `12px` (runtime var --thumb-size) · background-color `#ffffff` / `#141414` (background) · box-shadow `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| display | `block` | `block` | `block` |
| aspect-ratio | `1 / 1` | `1 / 1` | `aspect-square` |
| height | `100%` | `100%` | `h-full` |
| transform-origin | `0` | `0` | `origin-left` |
| border-radius | `12px` | `runtime var --thumb-size` | `rounded-(--thumb-size)` |
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |
| box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-sm/5` |
| will-change | `transform` | `transform` | `will-change-transform` |
| transition | `translate 150ms, border-radius 150ms, scale 100ms 100ms, transform-origin 150ms` | `translate .15s,border-radius .15s,scale .1s .1s,transform-origin .15s` | `[transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=drawer-menu-checkbox-item][data-checked]))) | transform-origin | `12px 50%` | `in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:origin-[var(--thumb-size)_50%]` |
| in(:where(*:is([data-slot=drawer-menu-checkbox-item][data-checked]))) | translate | `8px 0` | `in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:translate-x-[calc(var(--thumb-size)-4px)]` |
| in(:where(*:is([data-slot=drawer-menu-checkbox-item]:active))) | border-radius | `12px/13.2px` | `in-[[data-slot=drawer-menu-checkbox-item]:active]:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.10)]` |
| in(:where(*:is([data-slot=drawer-menu-checkbox-item]:active))) | scale | `110% 1` | `in-[[data-slot=drawer-menu-checkbox-item]:active]:not-data-disabled:scale-x-110` |
| rtl | transform-origin | `100%` | `rtl:origin-right` |
| in(:where(*:is([data-slot=drawer-menu-checkbox-item][data-checked]))) + rtl | translate | `-8px 0` | `rtl:in-[[data-slot=drawer-menu-checkbox-item][data-checked]]:-translate-x-[calc(var(--thumb-size)-4px)]` |

**element — mobile differences**: border-radius `16px`

### `DrawerMenuCheckboxItem` → (no data-slot) · `<CheckboxPrimitive.Indicator>` (line 561)

Prop defaults: `variant="default"`

#### base

```txt
col-start-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |

### `DrawerMenuCheckboxItem` → (no data-slot) · `<span>` (line 576)

Prop defaults: `variant="default"`

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

### `DrawerMenuRadioGroup` → `[data-slot=drawer-menu-radio-group]` · `<RadioGroupPrimitive>` (line 588)

#### base

```txt
flex flex-col
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |

### `DrawerMenuRadioItem` → `[data-slot=drawer-menu-radio-item]` · `<RadioPrimitive.Root>` (line 608)

Attributes: `disabled={disabled}`, `render={render}`, `value={value}`

#### base

```txt
grid min-h-9 w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none hover:bg-accent hover:text-accent-foreground data-disabled:pointer-events-none data-disabled:opacity-64 sm:min-h-8 sm:text-sm [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0 grid-cols-[1rem_minmax(0,1fr)] items-center pe-4
```

**element — default**

Key metrics: display `grid` · min-height `32px` · width `100%` · padding-inline `8px` · padding-block `4px` · padding-inline-end `16px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| min-height | `32px` | `calc(var(--spacing) * 8)` | `sm:min-h-8` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| grid-template-columns | `16px minmax(0,1fr)` | `1rem minmax(0,1fr)` | `grid-cols-[1rem_minmax(0,1fr)]` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| padding-inline-end | `16px` | `calc(var(--spacing) * 4)` | `pe-4` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: min-height `36px` · font-size `16px` · line-height `24px`

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-mx-0.5` |
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

### `DrawerMenuRadioItem` → (no data-slot) · `<RadioPrimitive.Indicator>` (line 620)

#### base

```txt
col-start-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |

### `DrawerMenuRadioItem` → (no data-slot) · `<span>` (line 635)

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

