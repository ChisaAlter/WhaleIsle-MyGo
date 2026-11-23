# Toggle

Source: `apps/ui/registry/default/ui/toggle.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/toggle.mdx` · Base UI: https://base-ui.com/react/components/toggle#api-reference

> A two-state button that can be toggled on or off.

Depends on: `@base-ui/react/toggle`

Exports: `Toggle`

## Behavior & keyboard

- Two-state button (`aria-pressed`); `Enter`/`Space` toggles; `data-pressed` when on. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `toggleVariants` (line 8)

Defaults: {"size":"default","variant":"default"}

```txt
base: relative inline-flex shrink-0 cursor-pointer select-none items-center justify-center gap-2 whitespace-nowrap rounded-lg border font-medium text-base text-foreground outline-none transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 data-pressed:bg-input/64 data-pressed:text-accent-foreground sm:text-sm [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0
size.default: h-9 min-w-9 px-[calc(--spacing(2)-1px)] sm:h-8 sm:min-w-8
size.lg: h-10 min-w-10 px-[calc(--spacing(2.5)-1px)] sm:h-9 sm:min-w-9
size.sm: h-8 min-w-8 px-[calc(--spacing(1.5)-1px)] sm:h-7 sm:min-w-7
variant.default: border-transparent
variant.outline: border-input bg-background not-dark:bg-clip-padding shadow-xs/5 not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] dark:bg-input/32 dark:data-pressed:bg-input dark:hover:bg-input/64 dark:not-disabled:not-active:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] dark:not-disabled:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/2%)] [:disabled,:active,[data-pressed]]:shadow-none
```

## Parts

### `Toggle` → `[data-slot=toggle]` · `<TogglePrimitive>` (line 38)

#### base (cva defaults: size=default, variant=default)

```txt
relative inline-flex shrink-0 cursor-pointer select-none items-center justify-center gap-2 whitespace-nowrap rounded-lg border font-medium text-base text-foreground outline-none transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 data-pressed:bg-input/64 data-pressed:text-accent-foreground sm:text-sm [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0 h-9 min-w-9 px-[calc(--spacing(2)-1px)] sm:h-8 sm:min-w-8 border-transparent
```

**element — default**

Key metrics: display `inline-flex` · height `32px` · min-width `32px` · padding-inline `7px` · gap `8px` · border-width `1px` · border-color `#00000000` · border-radius `10px` (radius) · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium) · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `32px` | `calc(var(--spacing) * 8)` | `sm:h-8` |
| min-width | `32px` | `calc(var(--spacing) * 8)` | `sm:min-w-8` |
| flex-shrink | `0` | `0` | `shrink-0` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding-inline | `7px` | `calc(calc(var(--spacing) * 2) - 1px)` | `px-[calc(--spacing(2)-1px)]` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| pressed | background-color | `#00000010` / `#ffffff0d` | `data-pressed:bg-input/64` |

**element — mobile differences**: height `36px` · min-width `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |

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

#### toggleVariants: size=lg

```txt
h-10 min-w-10 px-[calc(--spacing(2.5)-1px)] sm:h-9 sm:min-w-9
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `36px` | `calc(var(--spacing) * 9)` | `sm:h-9` |
| min-width | `36px` | `calc(var(--spacing) * 9)` | `sm:min-w-9` |
| padding-inline | `9px` | `calc(calc(var(--spacing) * 2.5) - 1px)` | `px-[calc(--spacing(2.5)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| pressed | background-color | `#00000010` / `#ffffff0d` | `data-pressed:bg-input/64` |

**element — mobile differences**: height `40px` · min-width `40px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### toggleVariants: size=sm

```txt
h-8 min-w-8 px-[calc(--spacing(1.5)-1px)] sm:h-7 sm:min-w-7
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `28px` | `calc(var(--spacing) * 7)` | `sm:h-7` |
| min-width | `28px` | `calc(var(--spacing) * 7)` | `sm:min-w-7` |
| padding-inline | `5px` | `calc(calc(var(--spacing) * 1.5) - 1px)` | `px-[calc(--spacing(1.5)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| pressed | background-color | `#00000010` / `#ffffff0d` | `data-pressed:bg-input/64` |

**element — mobile differences**: height `32px` · min-width `32px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### toggleVariants: variant=outline

```txt
border-input bg-background not-dark:bg-clip-padding shadow-xs/5 not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] dark:bg-input/32 dark:data-pressed:bg-input dark:hover:bg-input/64 dark:not-disabled:not-active:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] dark:not-disabled:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/2%)] [:disabled,:active,[data-pressed]]:shadow-none
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0d` | `hover:bg-accent` |
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #0000000d` / `0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #0000000d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| pressed | background-color | `#00000010` / `#ffffff14` | `data-pressed:bg-input/64` |
| pressed | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `36px` · min-width `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

