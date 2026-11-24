# Scroll Area

Source: `apps/ui/registry/default/ui/scroll-area.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/scroll-area.mdx` · Base UI: https://base-ui.com/react/components/scroll-area#api-reference

> A native scroll container with custom scrollbars.

Depends on: `@base-ui/react/scroll-area`

Exports: `ScrollArea`, `ScrollBar`

## Behavior & keyboard

- Custom scrollbars overlaying native scroll; keyboard scrolling is native on the viewport; scrollbars appear on hover/scroll (`data-hovering`, `data-scrolling`). _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ScrollArea` → (no data-slot) · `<ScrollAreaPrimitive.Root>` (line 24)

Prop defaults: `scrollFade=false`, `scrollbarGutter=false`, `fill=false`, `clampContentMinWidth=true`, `overscrollContain=false`

#### base

```txt
size-full min-h-0
```

**element — default**

Key metrics: height `100%` · min-height `0px` · width `100%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |

### `ScrollArea` → `[data-slot=scroll-area-viewport]` · `<ScrollAreaPrimitive.Viewport>` (line 28)

Prop defaults: `scrollFade=false`, `scrollbarGutter=false`, `fill=false`, `clampContentMinWidth=true`, `overscrollContain=false`

#### base

```txt
h-full rounded-[inherit] outline-none transition-shadow focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background
```

**element — default**

Key metrics: height `100%` · border-radius `inherit` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `100%` | `100%` | `h-full` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |

#### when `overscrollContain`

```txt
data-has-overflow-y:overscroll-y-contain data-has-overflow-x:overscroll-x-contain
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| [data-has-overflow-y] | overscroll-behavior-y | `contain` | `data-has-overflow-y:overscroll-y-contain` |
| [data-has-overflow-x] | overscroll-behavior-x | `contain` | `data-has-overflow-x:overscroll-x-contain` |

#### when `scrollFade`

```txt
mask-t-from-[calc(100%-min(var(--fade-size),var(--scroll-area-overflow-y-start)))] mask-b-from-[calc(100%-min(var(--fade-size),var(--scroll-area-overflow-y-end)))] mask-l-from-[calc(100%-min(var(--fade-size),var(--scroll-area-overflow-x-start)))] mask-r-from-[calc(100%-min(var(--fade-size),var(--scroll-area-overflow-x-end)))] [--fade-size:1.5rem]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| mask-image | `linear-gradient(to left, black calc(100% - min(24px, var(--scroll-area-overflow-x-start))), transparent 100%), linear-gradient(to right, black calc(100% - min(24px, var(--scroll-area-overflow-x-end))), transparent 100%), linear-gradient(to bottom, black calc(100% - min(24px, var(--scroll-area-overflow-y-end))), transparent 100%), linear-gradient(to top, black calc(100% - min(24px, var(--scroll-area-overflow-y-start))), transparent 100%), linear-gradient(#fff, #fff), linear-gradient(#fff, #fff)` | `var(--tw-mask-linear), var(--tw-mask-radial), var(--tw-mask-conic)` | `mask-l-from-[calc(100%-min(var(--fade-size),var(--scroll-area-overflow-x-start)))]` |
| mask-composite | `intersect` | `intersect` | `mask-l-from-[calc(100%-min(var(--fade-size),var(--scroll-area-overflow-x-start)))]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |

#### when `scrollbarGutter`

```txt
data-has-overflow-y:pe-2.5 data-has-overflow-x:pb-2.5
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| [data-has-overflow-y] | padding-inline-end | `10px` | `data-has-overflow-y:pe-2.5` |
| [data-has-overflow-x] | padding-bottom | `10px` | `data-has-overflow-x:pb-2.5` |

### `ScrollArea` → `[data-slot=scroll-area-content]` · `<ScrollAreaPrimitive.Content>` (line 40)

Attributes: `style={clampContentMinWidth ? { minWidth: 0 } : undefined}`

Prop defaults: `scrollFade=false`, `scrollbarGutter=false`, `fill=false`, `clampContentMinWidth=true`, `overscrollContain=false`

#### when `fill`

```txt
size-full
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |

### `ScrollArea` → `[data-slot=scroll-area-corner]` · `<ScrollAreaPrimitive.Corner>` (line 50)

Prop defaults: `scrollFade=false`, `scrollbarGutter=false`, `fill=false`, `clampContentMinWidth=true`, `overscrollContain=false`

### `ScrollBar` → `[data-slot=scroll-area-scrollbar]` · `<ScrollAreaPrimitive.Scrollbar>` (line 61)

Attributes: `orientation={orientation}`

Prop defaults: `orientation="vertical"`

#### base

```txt
m-1 flex opacity-0 transition-opacity delay-300 data-[orientation=horizontal]:h-1.5 data-[orientation=vertical]:w-1.5 data-[orientation=horizontal]:flex-col data-hovering:opacity-100 data-scrolling:opacity-100 data-hovering:delay-0 data-scrolling:delay-0 data-hovering:duration-100 data-scrolling:duration-100
```

**element — default**

Key metrics: display `flex` · opacity `0%` · transition-property `opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `4px` | `calc(var(--spacing) * 1)` | `m-1` |
| display | `flex` | `flex` | `flex` |
| opacity | `0%` | `0%` | `opacity-0` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-opacity` |
| transition-delay | `300ms` | `300ms` | `delay-300` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | height | `6px` | `data-[orientation=horizontal]:h-1.5` |
| [data-orientation=horizontal] | flex-direction | `column` | `data-[orientation=horizontal]:flex-col` |
| [data-orientation=vertical] | width | `6px` | `data-[orientation=vertical]:w-1.5` |
| [data-hovering] | opacity | `100%` | `data-hovering:opacity-100` |
| [data-hovering] | transition-duration | `100ms` | `data-hovering:duration-100` |
| [data-hovering] | transition-delay | `0ms` | `data-hovering:delay-0` |
| [data-scrolling] | opacity | `100%` | `data-scrolling:opacity-100` |
| [data-scrolling] | transition-duration | `100ms` | `data-scrolling:duration-100` |
| [data-scrolling] | transition-delay | `0ms` | `data-scrolling:delay-0` |

### `ScrollBar` → `[data-slot=scroll-area-thumb]` · `<ScrollAreaPrimitive.Thumb>` (line 70)

Prop defaults: `orientation="vertical"`

#### base

```txt
relative flex-1 rounded-full bg-foreground/20
```

**element — default**

Key metrics: border-radius `9999px` · background-color `#26262633` / `#f5f5f533` (foreground/20%)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| flex | `1` | `1` | `flex-1` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#26262633` / `#f5f5f533` | `foreground/20%` | `bg-foreground/20` |

