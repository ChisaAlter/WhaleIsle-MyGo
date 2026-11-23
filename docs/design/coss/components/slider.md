# Slider

Source: `apps/ui/registry/default/ui/slider.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/slider.mdx` · Base UI: https://base-ui.com/react/components/slider#api-reference

> An input where the user selects a value from within a given range.

Depends on: `@base-ui/react/slider`

Exports: `Slider`, `SliderValue`

## Behavior & keyboard

- Thumb(s) are focusable; `ArrowLeft/Right/Up/Down` step, `Shift+Arrow` large step, `PageUp/PageDown`, `Home`/`End` min/max; pointer drag sets `data-dragging`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Slider` → (no data-slot) · `<SliderPrimitive.Root>` (line 29)

Attributes: `defaultValue={defaultValue}`, `max={max}`, `min={min}`, `thumbAlignment=edge`, `value={value}`

Prop defaults: `min=0`, `max=100`

#### base

```txt
data-[orientation=horizontal]:w-full
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | width | `100%` | `data-[orientation=horizontal]:w-full` |

### `Slider` → `[data-slot=slider-control]` · `<SliderPrimitive.Control>` (line 39)

Prop defaults: `min=0`, `max=100`

#### base

```txt
flex touch-none select-none data-disabled:pointer-events-none data-[orientation=vertical]:h-full data-[orientation=vertical]:min-h-44 data-[orientation=horizontal]:w-full data-[orientation=horizontal]:min-w-44 data-[orientation=vertical]:flex-col data-disabled:opacity-64
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| touch-action | `none` | `none` | `touch-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| [data-orientation=vertical] | height | `100%` | `data-[orientation=vertical]:h-full` |
| [data-orientation=vertical] | min-height | `176px` | `data-[orientation=vertical]:min-h-44` |
| [data-orientation=vertical] | flex-direction | `column` | `data-[orientation=vertical]:flex-col` |
| [data-orientation=horizontal] | width | `100%` | `data-[orientation=horizontal]:w-full` |
| [data-orientation=horizontal] | min-width | `176px` | `data-[orientation=horizontal]:min-w-44` |

### `Slider` → `[data-slot=slider-track]` · `<SliderPrimitive.Track>` (line 43)

Prop defaults: `min=0`, `max=100`

#### base

```txt
relative grow select-none before:absolute before:rounded-full before:bg-input data-[orientation=horizontal]:h-1 data-[orientation=vertical]:h-full data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-1 data-[orientation=horizontal]:before:inset-x-0.5 data-[orientation=vertical]:before:inset-x-0 data-[orientation=horizontal]:before:inset-y-0 data-[orientation=vertical]:before:inset-y-0.5
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| flex-grow | `1` | `1` | `grow` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | height | `4px` | `data-[orientation=horizontal]:h-1` |
| [data-orientation=horizontal] | width | `100%` | `data-[orientation=horizontal]:w-full` |
| [data-orientation=vertical] | height | `100%` | `data-[orientation=vertical]:h-full` |
| [data-orientation=vertical] | width | `4px` | `data-[orientation=vertical]:w-1` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:bg-input` |
| position | `absolute` | `absolute` | `before:absolute` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `before:bg-input` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | inset-inline | `2px` | `data-[orientation=horizontal]:before:inset-x-0.5` |
| [data-orientation=horizontal] | inset-block | `0px` | `data-[orientation=horizontal]:before:inset-y-0` |
| [data-orientation=vertical] | inset-inline | `0px` | `data-[orientation=vertical]:before:inset-x-0` |
| [data-orientation=vertical] | inset-block | `2px` | `data-[orientation=vertical]:before:inset-y-0.5` |

### `Slider` → `[data-slot=slider-indicator]` · `<SliderPrimitive.Indicator>` (line 47)

Prop defaults: `min=0`, `max=100`

#### base

```txt
select-none rounded-full bg-primary data-[orientation=horizontal]:ms-0.5 data-[orientation=vertical]:mb-0.5
```

**element — default**

Key metrics: border-radius `9999px` · background-color `#262626` / `#f5f5f5` (primary)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#262626` / `#f5f5f5` | `primary` | `bg-primary` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | margin-inline-start | `2px` | `data-[orientation=horizontal]:ms-0.5` |
| [data-orientation=vertical] | margin-bottom | `2px` | `data-[orientation=vertical]:mb-0.5` |

### `Slider` → `[data-slot=slider-thumb]` · `<SliderPrimitive.Thumb>` (line 52)

Attributes: `getAriaLabel={getAriaLabel}`, `index={index}`

Prop defaults: `min=0`, `max=100`

#### base

```txt
block size-5 shrink-0 select-none rounded-full border border-input bg-white not-dark:bg-clip-padding shadow-xs/5 outline-none transition-[box-shadow,scale] before:absolute before:inset-0 before:rounded-full before:shadow-[0_1px_--theme(--color-black/4%)] has-focus-visible:ring-[3px] has-focus-visible:ring-ring/24 data-dragging:scale-120 sm:size-4 dark:border-background dark:has-focus-visible:ring-ring/48 [:has(*:focus-visible),[data-dragging]]:shadow-none
```

**element — default**

Key metrics: display `block` · height `16px` · width `16px` · border-width `1px` · border-color `#0000001a` / `#141414` (input) · border-radius `9999px` · background-color `#ffffff` (color-white) · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow, scale` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#141414` | `input` | `border-input` |
| display | `block` | `block` | `block` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| flex-shrink | `0` | `0` | `shrink-0` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#ffffff` | `color-white` | `bg-white` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow, scale` | `box-shadow,scale` | `transition-[box-shadow,scale]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[box-shadow,scale]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[box-shadow,scale]` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373737a` | `[:has(*:focus-visible),[data-dragging]]:shadow-none` |
| [data-dragging] | box-shadow | `none` | `[:has(*:focus-visible),[data-dragging]]:shadow-none` |
| [data-dragging] | scale | `120% 120%` | `data-dragging:scale-120` |

**element — mobile differences**: width `20px` · height `20px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |
| box-shadow | `0 1px #0000000a` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `SliderValue` → `[data-slot=slider-value]` · `<SliderPrimitive.Value>` (line 71)

#### base

```txt
flex justify-end text-sm
```

**element — default**

Key metrics: display `flex` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| justify-content | `flex-end` | `flex-end` | `justify-end` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |

