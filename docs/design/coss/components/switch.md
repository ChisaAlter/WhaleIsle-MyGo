# Switch

Source: `apps/ui/registry/default/ui/switch.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/switch.mdx` · Base UI: https://base-ui.com/react/components/switch#api-reference

> A control that indicates whether a setting is on or off.

Depends on: `@base-ui/react/switch`

Exports: `Switch`

## Behavior & keyboard

- `role=switch` button; `Space`/`Enter` toggles; thumb translates on `data-checked`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Switch` → `[data-slot=switch]` · `<SwitchPrimitive.Root>` (line 12)

#### base

```txt
inline-flex h-[calc(var(--thumb-size)+2px)] w-[calc(var(--thumb-size)*2-2px)] shrink-0 items-center rounded-full p-px outline-none transition-[background-color,box-shadow] duration-200 [--thumb-size:--spacing(5)] focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background data-disabled:cursor-not-allowed data-checked:bg-primary data-unchecked:bg-input data-disabled:opacity-64 sm:[--thumb-size:--spacing(4)]
```

**element — default**

Key metrics: display `inline-flex` · height `18px` · width `30px` · padding `1px` · border-radius `9999px` · transition-property `background-color, box-shadow` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `18px` | `calc(var(--thumb-size) + 2px)` | `h-[calc(var(--thumb-size)+2px)]` |
| width | `30px` | `calc(var(--thumb-size) * 2 - 2px)` | `w-[calc(var(--thumb-size)*2-2px)]` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| padding | `1px` | `1px` | `p-px` |
| transition-property | `background-color, box-shadow` | `background-color,box-shadow` | `transition-[background-color,box-shadow]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[background-color,box-shadow]` |
| transition-duration | `200ms` | `200ms` | `duration-200` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | cursor | `not-allowed` | `data-disabled:cursor-not-allowed` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| checked | background-color | `#262626` / `#f5f5f5` | `data-checked:bg-primary` |
| unchecked | background-color | `#0000001a` / `#ffffff14` | `data-unchecked:bg-input` |

**element — mobile differences**: height `22px` · width `38px`

### `Switch` → `[data-slot=switch-thumb]` · `<SwitchPrimitive.Thumb>` (line 20)

#### base

```txt
pointer-events-none block aspect-square h-full origin-left in-[[role=switch]:active,[data-slot=label]:active,[data-slot=field-label]:active]:not-data-disabled:scale-x-110 in-[[role=switch]:active,[data-slot=label]:active,[data-slot=field-label]:active]:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.1)] rounded-(--thumb-size) bg-background shadow-sm/5 will-change-transform [transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s] data-checked:origin-[var(--thumb-size)_50%] data-checked:translate-x-[calc(var(--thumb-size)-4px)] rtl:origin-right rtl:data-checked:origin-left rtl:data-checked:-translate-x-[calc(var(--thumb-size)-4px)]
```

**element — default**

Key metrics: display `block` · height `100%` · border-radius `16px` (runtime var --thumb-size) · background-color `#ffffff` / `#141414` (background) · box-shadow `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| display | `block` | `block` | `block` |
| aspect-ratio | `1 / 1` | `1 / 1` | `aspect-square` |
| height | `100%` | `100%` | `h-full` |
| transform-origin | `0` | `0` | `origin-left` |
| border-radius | `16px` | `runtime var --thumb-size` | `rounded-(--thumb-size)` |
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |
| box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-sm/5` |
| will-change | `transform` | `transform` | `will-change-transform` |
| transition | `translate 150ms, border-radius 150ms, scale 100ms 100ms, transform-origin 150ms` | `translate .15s,border-radius .15s,scale .1s .1s,transform-origin .15s` | `[transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([role=switch]:active,[data-slot=label]:active,[data-slot=field-label]:active))) | border-radius | `16px/17.6px` | `in-[[role=switch]:active,[data-slot=label]:active,[data-slot=field-label]:active]:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.1)]` |
| in(:where(*:is([role=switch]:active,[data-slot=label]:active,[data-slot=field-label]:active))) | scale | `110% 1` | `in-[[role=switch]:active,[data-slot=label]:active,[data-slot=field-label]:active]:not-data-disabled:scale-x-110` |
| checked | transform-origin | `16px 50%` | `data-checked:origin-[var(--thumb-size)_50%]` |
| checked | translate | `12px 0` | `data-checked:translate-x-[calc(var(--thumb-size)-4px)]` |
| rtl | transform-origin | `100%` | `rtl:origin-right` |
| checked + rtl | translate | `-12px 0` | `rtl:data-checked:-translate-x-[calc(var(--thumb-size)-4px)]` |

**element — mobile differences**: border-radius `20px`

