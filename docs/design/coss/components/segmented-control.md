# Segmented Control

Source: `apps/ui/registry/default/ui/segmented-control.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/segmented-control.mdx`

> A visual pattern for presenting related choices, navigation destinations, filters, or content views.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `SegmentedControl`, `SegmentedControlItem`, `isSameRect`, `getCssSize`, `measureItem`, `SegmentedControlIndicator`

Module constants: `segmentedControlItemLayoutClassName = gap-1.5 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0`, `segmentedControlIndicatorSurfaceClassName = rounded-md bg-background shadow-sm/5 dark:bg-input`, `activeItemSelector = :scope > [data-checked], :scope > [aria-current="page"], :scope > [data-pressed]`, `MAX_LAYOUT_ROUNDING_ERROR = 2`

## Behavior & keyboard

- Single-select toggle group with a sliding indicator; arrow keys move focus (roving tabindex), `Enter`/`Space` select. _(inferred: built on Base UI ToggleGroup/Radio semantics; see source)_

Docs mentions:

- A segmented control is a visual pattern, not a standalone behavior. `SegmentedControl` and `SegmentedControlItem` provide the presentation and the sliding indicator; the element you pass to `render` provides the semantics, keyboard interactions, and state model.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `SegmentedControl` → `[data-slot=segmented-control]` · `useRender <div>` (line 37)

Attributes: `data-size={size}`

Prop defaults: `size="default"`

#### base

```txt
relative z-0 flex w-fit items-center justify-center gap-0.5 rounded-lg bg-muted p-0.5
```

**element — default**

Key metrics: display `flex` · width `fit-content` · padding `2px` · gap `2px` · border-radius `10px` (radius) · background-color `#0000000a` / `#ffffff0a` (muted) · z-index `0`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| z-index | `0` | `0` | `z-0` |
| display | `flex` | `flex` | `flex` |
| width | `fit-content` | `fit-content` | `w-fit` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-0.5` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#0000000a` / `#ffffff0a` | `muted` | `bg-muted` |
| padding | `2px` | `calc(var(--spacing) * 0.5)` | `p-0.5` |

### `SegmentedControlItem` → `[data-slot=segmented-control-item]` · `useRender <button>` (line 76)

Attributes: `data-size={resolvedSize}`

Dynamic className inputs (not statically resolvable): `segmentedControlItemSizeClassNames[resolvedSize]`

#### base

```txt
relative inline-flex shrink-0 cursor-pointer select-none items-center justify-center whitespace-nowrap rounded-md border border-transparent font-medium text-base text-muted-foreground/72 outline-2 outline-transparent transition-[color,outline-color] hover:bg-transparent hover:text-muted-foreground focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-64 data-disabled:pointer-events-none data-disabled:opacity-64 sm:text-sm data-checked:bg-background data-checked:text-foreground data-checked:shadow-sm/5 dark:data-checked:bg-input aria-[current=page]:bg-background aria-[current=page]:text-foreground aria-[current=page]:shadow-sm/5 dark:aria-[current=page]:bg-input data-pressed:bg-background data-pressed:text-foreground data-pressed:shadow-sm/5 dark:data-pressed:bg-input [:has(>[data-slot=segmented-control-indicator]:not([hidden]))>&]:bg-transparent! [:has(>[data-slot=segmented-control-indicator]:not([hidden]))>&]:shadow-none! gap-1.5 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · gap `6px` · border-width `1px` · border-color `#00000000` · border-radius `8px` · color `#686868b8` / `#818181b8` (muted-foreground/72%) · font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium) · transition-property `color, outline-color` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| outline-color | `#00000000` | `transparent` | `outline-transparent` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `text-muted-foreground/72` |
| outline-style | `solid` | `var(--tw-outline-style)` | `outline-2` |
| outline-width | `2px` | `2px` | `outline-2` |
| transition-property | `color, outline-color` | `color,outline-color` | `transition-[color,outline-color]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[color,outline-color]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[color,outline-color]` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#686868` / `#818181` | `hover:text-muted-foreground` |
| hover | background-color | `#00000000` | `hover:bg-transparent` |
| focus-visible | outline-color | `#a1a1a1` / `#737373` | `focus-visible:outline-ring` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| checked | color | `#262626` / `#f5f5f5` | `data-checked:text-foreground` |
| checked | background-color | `#ffffff` / `#ffffff14` | `data-checked:bg-background` |
| checked | box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `data-checked:shadow-sm/5` |
| selected | color | `#262626` / `#f5f5f5` | `aria-[current=page]:text-foreground` |
| selected | background-color | `#ffffff` / `#ffffff14` | `aria-[current=page]:bg-background` |
| selected | box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `aria-[current=page]:shadow-sm/5` |
| pressed | color | `#262626` / `#f5f5f5` | `data-pressed:text-foreground` |
| pressed | background-color | `#ffffff` / `#ffffff14` | `data-pressed:bg-background` |
| pressed | box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `data-pressed:shadow-sm/5` |
| parent(:has(>[data-slot=segmented-control-indicator]:not([hidden]))) | background-color | `#00000000` | `[:has(>[data-slot=segmented-control-indicator]:not([hidden]))>&]:bg-transparent!` |
| parent(:has(>[data-slot=segmented-control-indicator]:not([hidden]))) | box-shadow | `none` | `[:has(>[data-slot=segmented-control-indicator]:not([hidden]))>&]:shadow-none!` |

**element — mobile differences**: font-size `16px` · line-height `24px`

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

### `isSameRect`

No own classes (re-export / wrapper of the primitive).

### `getCssSize`

No own classes (re-export / wrapper of the primitive).

### `measureItem`

No own classes (re-export / wrapper of the primitive).

### `SegmentedControlIndicator` → `[data-slot=segmented-control-indicator]` · `<span>` (line 215)

Attributes: `aria-hidden=true`, `hidden={!rect}`

#### base

```txt
pointer-events-none absolute top-0 left-0 -z-1 h-(--active-item-height) w-(--active-item-width) translate-x-(--active-item-left) translate-y-(--active-item-top) transition-[width,height,translate] duration-200 ease-in-out motion-reduce:transition-none rounded-md bg-background shadow-sm/5 dark:bg-input
```

**element — default**

Key metrics: height `var(--active-item-height)` (runtime var --active-item-height) · width `var(--active-item-width)` (runtime var --active-item-width) · border-radius `8px` · background-color `#ffffff` / `#ffffff14` (background) · box-shadow `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` · z-index `-1` · transition-property `width, height, translate` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| top | `0px` | `calc(var(--spacing) * 0)` | `top-0` |
| left | `0px` | `calc(var(--spacing) * 0)` | `left-0` |
| z-index | `-1` | `calc(1 * -1)` | `-z-1` |
| height | `var(--active-item-height)` | `runtime var --active-item-height` | `h-(--active-item-height)` |
| width | `var(--active-item-width)` | `runtime var --active-item-width` | `w-(--active-item-width)` |
| translate | `var(--active-item-left) var(--active-item-top)` | `var(--tw-translate-x) var(--tw-translate-y)` | `translate-y-(--active-item-top)` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| background-color | `#ffffff` / `#ffffff14` | `background` | `bg-background` |
| box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-sm/5` |
| transition-property | `width, height, translate` | `width,height,translate` | `transition-[width,height,translate]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

## Other class strings in this file

Class strings not bound directly to a JSX `className` (lookup tables, maps, helpers). Resolved per class (desktop, light / dark, no state):

- line 14: context `default: "h-8.5 px-[calc(--spacing(2.5)-1px)] sm:h-7.5"`

  ```txt
  h-8.5 px-[calc(--spacing(2.5)-1px)] sm:h-7.5
  ```

  Resolved: height: `30px` · padding-inline: `9px`

- line 15: context `lg: "h-9.5 px-[calc(--spacing(3)-1px)] sm:h-8.5"`

  ```txt
  h-9.5 px-[calc(--spacing(3)-1px)] sm:h-8.5
  ```

  Resolved: height: `34px` · padding-inline: `11px`

- line 16: context `sm: "h-7.5 px-[calc(--spacing(2)-1px)] sm:h-6.5"`

  ```txt
  h-7.5 px-[calc(--spacing(2)-1px)] sm:h-6.5
  ```

  Resolved: height: `26px` · padding-inline: `7px`

