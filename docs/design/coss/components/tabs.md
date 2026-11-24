# Tabs

Source: `apps/ui/registry/default/ui/tabs.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/tabs.mdx` · Base UI: https://base-ui.com/react/components/tabs#api-reference

> A component for toggling between related panels on the same page.

Depends on: `@base-ui/react/tabs`, `@/ui/segmented-control`

Exports: `Tabs`, `TabsList`, `TabsTab`, `TabsPanel`

## Behavior & keyboard

- Tablist with roving focus: `ArrowLeft/Right` (horizontal) or `ArrowUp/Down` (vertical) move focus, activation on focus by default (Base UI `activateOnFocus` false → `Enter`/`Space` activates; verify), `Home`/`End`. Active indicator animates between tabs using `--active-tab-*` CSS vars. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Tabs` → `[data-slot=tabs]` · `<TabsPrimitive.Root>` (line 24)

#### base

```txt
flex flex-col gap-2 data-[orientation=vertical]:flex-row
```

**element — default**

Key metrics: display `flex` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=vertical] | flex-direction | `row` | `data-[orientation=vertical]:flex-row` |

### `TabsList` → `[data-slot=tabs-list]` · `<TabsPrimitive.List>` (line 46)

Attributes: `data-size={size}`

Prop defaults: `variant="default"`, `size="default"`

#### base

```txt
relative z-0 flex w-fit items-center justify-center gap-x-0.5 text-muted-foreground data-[orientation=vertical]:flex-col
```

**element — default**

Key metrics: display `flex` · width `fit-content` · column-gap `2px` · color `#686868` / `#818181` (muted-foreground) · z-index `0`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| z-index | `0` | `0` | `z-0` |
| display | `flex` | `flex` | `flex` |
| width | `fit-content` | `fit-content` | `w-fit` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| column-gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-x-0.5` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=vertical] | flex-direction | `column` | `data-[orientation=vertical]:flex-col` |

#### when `variant === "default"`

```txt
rounded-lg bg-muted p-0.5 text-muted-foreground/72
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `text-muted-foreground/72` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#0000000a` / `#ffffff0a` | `muted` | `bg-muted` |
| padding | `2px` | `calc(var(--spacing) * 0.5)` | `p-0.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=vertical] | flex-direction | `column` | `data-[orientation=vertical]:flex-col` |

#### when `!(variant === "default")`

```txt
data-[orientation=vertical]:px-1 data-[orientation=horizontal]:py-1 *:data-[slot=tabs-tab]:hover:bg-accent
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=vertical] | flex-direction | `column` | `data-[orientation=vertical]:flex-col` |
| [data-orientation=vertical] | padding-inline | `4px` | `data-[orientation=vertical]:px-1` |
| [data-orientation=horizontal] | padding-block | `4px` | `data-[orientation=horizontal]:py-1` |

**`> *[data-slot="tabs-tab"]:hover` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#0000000a` / `#ffffff0a` | `accent` | `*:data-[slot=tabs-tab]:hover:bg-accent` |

**`> *[data-slot="tabs-tab"]:hover` — mobile differences**: background-color `(unset)`

### `TabsList` → `[data-slot=tab-indicator]` · `<TabsPrimitive.Indicator>` (line 62)

Prop defaults: `variant="default"`, `size="default"`

Dynamic className inputs (not statically resolvable): `segmentedControlIndicatorSurfaceClassName` when `!(variant === "underline")`

#### base

```txt
absolute bottom-0 left-0 h-(--active-tab-height) w-(--active-tab-width) translate-x-(--active-tab-left) -translate-y-(--active-tab-bottom) transition-[width,translate] duration-200 ease-in-out motion-reduce:transition-none
```

**element — default**

Key metrics: height `var(--active-tab-height)` (runtime var --active-tab-height) · width `var(--active-tab-width)` (runtime var --active-tab-width) · transition-property `width, translate` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| bottom | `0px` | `calc(var(--spacing) * 0)` | `bottom-0` |
| left | `0px` | `calc(var(--spacing) * 0)` | `left-0` |
| height | `var(--active-tab-height)` | `runtime var --active-tab-height` | `h-(--active-tab-height)` |
| width | `var(--active-tab-width)` | `runtime var --active-tab-width` | `w-(--active-tab-width)` |
| translate | `var(--active-tab-left) calc(var(--active-tab-bottom) * -1)` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-y-(--active-tab-bottom)` |
| transition-property | `width, translate` | `width,translate` | `transition-[width,translate]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

#### when `variant === "underline"`

```txt
z-10 bg-primary data-[orientation=horizontal]:h-0.5 data-[orientation=vertical]:w-0.5 data-[orientation=vertical]:-translate-x-px data-[orientation=horizontal]:translate-y-px
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `10` | `10` | `z-10` |
| background-color | `#262626` / `#f5f5f5` | `primary` | `bg-primary` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | height | `2px` | `data-[orientation=horizontal]:h-0.5` |
| [data-orientation=horizontal] | translate | `var(--active-tab-left) 1px` | `data-[orientation=horizontal]:translate-y-px` |
| [data-orientation=vertical] | width | `2px` | `data-[orientation=vertical]:w-0.5` |
| [data-orientation=vertical] | translate | `-1px calc(var(--active-tab-bottom) * -1)` | `data-[orientation=vertical]:-translate-x-px` |

#### when `!(variant === "underline")`

```txt
-z-1
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `-1` | `calc(1 * -1)` | `-z-1` |

### `TabsTab` → `[data-slot=tabs-tab]` · `<TabsPrimitive.Tab>` (line 86)

Attributes: `data-size={resolvedSize}`

Dynamic className inputs (not statically resolvable): `segmentedControlItemLayoutClassName`, `segmentedControlItemSizeClassNames[resolvedSize]`

#### base

```txt
relative flex shrink-0 grow cursor-pointer items-center justify-center whitespace-nowrap rounded-md border border-transparent font-medium text-base outline-none transition-[color,background-color,box-shadow] hover:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring data-disabled:pointer-events-none data-[orientation=vertical]:w-full data-[orientation=vertical]:justify-start data-active:text-foreground data-disabled:opacity-64 sm:text-sm
```

**element — default**

Key metrics: display `flex` · border-width `1px` · border-color `#00000000` · border-radius `8px` · font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium) · transition-property `color, background-color, box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| flex-grow | `1` | `1` | `grow` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| transition-property | `color, background-color, box-shadow` | `color,background-color,box-shadow` | `transition-[color,background-color,box-shadow]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[color,background-color,box-shadow]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[color,background-color,box-shadow]` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#686868` / `#818181` | `hover:text-muted-foreground` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` / `0 0 0 2px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| [data-orientation=vertical] | justify-content | `flex-start` | `data-[orientation=vertical]:justify-start` |
| [data-orientation=vertical] | width | `100%` | `data-[orientation=vertical]:w-full` |
| selected | color | `#262626` / `#f5f5f5` | `data-active:text-foreground` |

**element — mobile differences**: font-size `16px` · line-height `24px`

### `TabsPanel` → `[data-slot=tabs-content]` · `<TabsPrimitive.Panel>` (line 105)

#### base

```txt
flex-1 outline-none
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex | `1` | `1` | `flex-1` |
| outline-style | `none` | `none` | `outline-none` |

