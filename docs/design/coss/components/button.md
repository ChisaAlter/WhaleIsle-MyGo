# Button

Source: `apps/ui/registry/default/ui/button.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/button.mdx`

> A button or a component that looks like a button.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`, `@/ui/spinner`

Exports: `Button`

## Behavior & keyboard

- Native `<button type=button>` (or `render` element). `Enter`/`Space` activate. `loading` sets `aria-disabled`, `data-loading`, disables the button and overlays a Spinner while text becomes transparent.
- Touch (`pointer: coarse`) adds an invisible `::after` hit area min 44×44px.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `buttonVariants` (line 10)

Defaults: {"size":"default","variant":"default"}

```txt
base: relative inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-lg border font-medium text-base outline-none transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 data-loading:select-none data-loading:text-transparent sm:text-sm [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0
size.default: h-9 px-[calc(--spacing(3)-1px)] sm:h-8
size.icon: size-9 sm:size-8
size.icon-lg: size-10 sm:size-9
size.icon-sm: size-8 sm:size-7
size.icon-xl: size-11 sm:size-10 [&_svg:not([class*='size-'])]:size-5 sm:[&_svg:not([class*='size-'])]:size-4.5
size.icon-xs: size-7 rounded-md before:rounded-[calc(var(--radius-md)-1px)] sm:size-6 not-in-data-[slot=input-group]:[&_svg:not([class*='size-'])]:size-4 sm:not-in-data-[slot=input-group]:[&_svg:not([class*='size-'])]:size-3.5
size.lg: h-10 px-[calc(--spacing(3.5)-1px)] sm:h-9
size.sm: h-8 gap-1.5 px-[calc(--spacing(2.5)-1px)] sm:h-7
size.xl: h-11 px-[calc(--spacing(4)-1px)] text-lg sm:h-10 sm:text-base [&_svg:not([class*='size-'])]:size-5 sm:[&_svg:not([class*='size-'])]:size-4.5
size.xs: h-7 gap-1 rounded-md px-[calc(--spacing(2)-1px)] text-sm before:rounded-[calc(var(--radius-md)-1px)] sm:h-6 sm:text-xs [&_svg:not([class*='size-'])]:size-4 sm:[&_svg:not([class*='size-'])]:size-3.5
variant.default: not-disabled:inset-shadow-[0_1px_--theme(--color-white/16%)] border-primary bg-primary text-primary-foreground shadow-primary/24 shadow-xs hover:bg-primary/90 data-pressed:bg-primary/90 *:data-[slot=button-loading-indicator]:text-primary-foreground [:active,[data-pressed]]:inset-shadow-[0_1px_--theme(--color-black/8%)] [:disabled,:active,[data-pressed]]:shadow-none
variant.destructive: not-disabled:inset-shadow-[0_1px_--theme(--color-white/16%)] border-destructive bg-destructive text-white shadow-destructive/24 shadow-xs hover:bg-destructive/90 data-pressed:bg-destructive/90 *:data-[slot=button-loading-indicator]:text-white [:active,[data-pressed]]:inset-shadow-[0_1px_--theme(--color-black/8%)] [:disabled,:active,[data-pressed]]:shadow-none
variant.destructive-outline: border-input bg-popover not-dark:bg-clip-padding text-destructive-foreground shadow-xs/5 not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] hover:border-destructive/32 hover:bg-destructive/4 data-pressed:border-destructive/32 data-pressed:bg-destructive/4 *:data-[slot=button-loading-indicator]:text-foreground dark:bg-input/32 dark:not-disabled:before:shadow-[0_-1px_--theme(--color-white/2%)] dark:not-disabled:not-active:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [:disabled,:active,[data-pressed]]:shadow-none
variant.ghost: border-transparent text-foreground hover:bg-accent data-pressed:bg-accent *:data-[slot=button-loading-indicator]:text-foreground
variant.link: border-transparent text-foreground underline-offset-4 hover:underline data-pressed:underline *:data-[slot=button-loading-indicator]:text-foreground
variant.outline: border-input bg-popover not-dark:bg-clip-padding text-foreground shadow-xs/5 not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] hover:bg-accent/50 data-pressed:bg-accent/50 *:data-[slot=button-loading-indicator]:text-foreground dark:bg-input/32 dark:data-pressed:bg-input/64 dark:hover:bg-input/64 dark:not-disabled:before:shadow-[0_-1px_--theme(--color-white/2%)] dark:not-disabled:not-active:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [:disabled,:active,[data-pressed]]:shadow-none
variant.secondary: border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/90 data-pressed:bg-secondary/90 *:data-[slot=button-loading-indicator]:text-secondary-foreground [:active,[data-pressed]]:bg-secondary/80
```

## Parts

### `Button` → `[data-slot=button]` · `useRender <button>` (line 71)

Attributes: `aria-disabled={loading \|\| undefined}`, `data-loading={loading ? "" : undefined}`, `disabled={isDisabled}`, `type={typeValue}`

Prop defaults: `loading=false`

#### base (cva defaults: size=default, variant=default)

```txt
relative inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-lg border font-medium text-base outline-none transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 data-loading:select-none data-loading:text-transparent sm:text-sm [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0 h-9 px-[calc(--spacing(3)-1px)] sm:h-8 not-disabled:inset-shadow-[0_1px_--theme(--color-white/16%)] border-primary bg-primary text-primary-foreground shadow-primary/24 shadow-xs hover:bg-primary/90 data-pressed:bg-primary/90 *:data-[slot=button-loading-indicator]:text-primary-foreground [:active,[data-pressed]]:inset-shadow-[0_1px_--theme(--color-black/8%)] [:disabled,:active,[data-pressed]]:shadow-none
```

**element — default**

Key metrics: display `inline-flex` · height `32px` · padding-inline `11px` · gap `8px` · border-width `1px` · border-color `#262626` / `#f5f5f5` (primary) · border-radius `10px` (radius) · background-color `#262626` / `#f5f5f5` (primary) · color `#fafafa` / `#262626` (primary-foreground) · font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium) · box-shadow `inset 0 1px #ffffff29, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 1px 2px 0 #f5f5f53d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#262626` / `#f5f5f5` | `primary` | `border-primary` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `32px` | `calc(var(--spacing) * 8)` | `sm:h-8` |
| flex-shrink | `0` | `0` | `shrink-0` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#262626` / `#f5f5f5` | `primary` | `bg-primary` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| color | `#fafafa` / `#262626` | `primary-foreground` | `text-primary-foreground` |
| box-shadow | `inset 0 1px #ffffff29, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 1px 2px 0 #f5f5f53d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-disabled:inset-shadow-[0_1px_--theme(--color-white/16%)]` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |

**`> *[data-slot="button-loading-indicator"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#fafafa` / `#262626` | `primary-foreground` | `*:data-[slot=button-loading-indicator]:text-primary-foreground` |

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

#### buttonVariants: size=icon

```txt
size-9 sm:size-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `(unset)` | `` | `(removed)` |
| width | `32px` | `calc(var(--spacing) * 8)` | `sm:size-8` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: width `36px` · height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: size=icon-lg

```txt
size-10 sm:size-9
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `36px` | `calc(var(--spacing) * 9)` | `sm:size-9` |
| padding-inline | `(unset)` | `` | `(removed)` |
| width | `36px` | `calc(var(--spacing) * 9)` | `sm:size-9` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: width `40px` · height `40px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: size=icon-sm

```txt
size-8 sm:size-7
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `28px` | `calc(var(--spacing) * 7)` | `sm:size-7` |
| padding-inline | `(unset)` | `` | `(removed)` |
| width | `28px` | `calc(var(--spacing) * 7)` | `sm:size-7` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: width `32px` · height `32px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: size=icon-xl

```txt
size-11 sm:size-10 [&_svg:not([class*='size-'])]:size-5 sm:[&_svg:not([class*='size-'])]:size-4.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `40px` | `calc(var(--spacing) * 10)` | `sm:size-10` |
| padding-inline | `(unset)` | `` | `(removed)` |
| width | `40px` | `calc(var(--spacing) * 10)` | `sm:size-10` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: width `44px` · height `44px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `18px` | `calc(var(--spacing) * 4.5)` | `sm:[&_svg:not([class*='size-'])]:size-4.5` |
| height | `18px` | `calc(var(--spacing) * 4.5)` | `sm:[&_svg:not([class*='size-'])]:size-4.5` |

**`svg:not([class*='size-'])` — mobile differences**: width `20px` · height `20px`

#### buttonVariants: size=icon-xs

```txt
size-7 rounded-md before:rounded-[calc(var(--radius-md)-1px)] sm:size-6 not-in-data-[slot=input-group]:[&_svg:not([class*='size-'])]:size-4 sm:not-in-data-[slot=input-group]:[&_svg:not([class*='size-'])]:size-3.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `24px` | `calc(var(--spacing) * 6)` | `sm:size-6` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| padding-inline | `(unset)` | `` | `(removed)` |
| width | `24px` | `calc(var(--spacing) * 6)` | `sm:size-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: width `28px` · height `28px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |

**`svg:not([class*='size-'])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `14px` | `calc(var(--spacing) * 3.5)` | `sm:not-in-data-[slot=input-group]:[&_svg:not([class*='size-'])]:size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `sm:not-in-data-[slot=input-group]:[&_svg:not([class*='size-'])]:size-3.5` |

**`svg:not([class*='size-'])` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-slot=input-group]) | width | `16px` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| in([data-slot=input-group]) | height | `16px` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `16px` · height `16px`

#### buttonVariants: size=lg

```txt
h-10 px-[calc(--spacing(3.5)-1px)] sm:h-9
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `36px` | `calc(var(--spacing) * 9)` | `sm:h-9` |
| padding-inline | `13px` | `calc(calc(var(--spacing) * 3.5) - 1px)` | `px-[calc(--spacing(3.5)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `40px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: size=sm

```txt
h-8 gap-1.5 px-[calc(--spacing(2.5)-1px)] sm:h-7
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `28px` | `calc(var(--spacing) * 7)` | `sm:h-7` |
| padding-inline | `9px` | `calc(calc(var(--spacing) * 2.5) - 1px)` | `px-[calc(--spacing(2.5)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `32px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: size=xl

```txt
h-11 px-[calc(--spacing(4)-1px)] text-lg sm:h-10 sm:text-base [&_svg:not([class*='size-'])]:size-5 sm:[&_svg:not([class*='size-'])]:size-4.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `40px` | `calc(var(--spacing) * 10)` | `sm:h-10` |
| padding-inline | `15px` | `calc(calc(var(--spacing) * 4) - 1px)` | `px-[calc(--spacing(4)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `44px` · font-size `18px` · line-height `28.001px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`svg:not([class*='size-'])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `18px` | `calc(var(--spacing) * 4.5)` | `sm:[&_svg:not([class*='size-'])]:size-4.5` |
| height | `18px` | `calc(var(--spacing) * 4.5)` | `sm:[&_svg:not([class*='size-'])]:size-4.5` |

**`svg:not([class*='size-'])` — mobile differences**: width `20px` · height `20px`

#### buttonVariants: size=xs

```txt
h-7 gap-1 rounded-md px-[calc(--spacing(2)-1px)] text-sm before:rounded-[calc(var(--radius-md)-1px)] sm:h-6 sm:text-xs [&_svg:not([class*='size-'])]:size-4 sm:[&_svg:not([class*='size-'])]:size-3.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `24px` | `calc(var(--spacing) * 6)` | `sm:h-6` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| padding-inline | `7px` | `calc(calc(var(--spacing) * 2) - 1px)` | `px-[calc(--spacing(2)-1px)]` |
| font-size | `12px` | `text-xs` | `sm:text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `sm:text-xs` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #2626263d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #f5f5f53d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#262626e6` / `#f5f5f5e6` | `hover:bg-primary/90` |
| pressed | background-color | `#262626e6` / `#f5f5f5e6` | `data-pressed:bg-primary/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `28px` · font-size `14px` · line-height `20px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: variant=destructive

```txt
not-disabled:inset-shadow-[0_1px_--theme(--color-white/16%)] border-destructive bg-destructive text-white shadow-destructive/24 shadow-xs hover:bg-destructive/90 data-pressed:bg-destructive/90 *:data-[slot=button-loading-indicator]:text-white [:active,[data-pressed]]:inset-shadow-[0_1px_--theme(--color-black/8%)] [:disabled,:active,[data-pressed]]:shadow-none
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#fb2c36` / `#fb414a` | `destructive` | `border-destructive` |
| background-color | `#fb2c36` / `#fb414a` | `destructive` | `bg-destructive` |
| color | `#ffffff` | `color-white` | `text-white` |
| box-shadow | `inset 0 1px #ffffff29, 0 1px 2px 0 #fb2c363d` / `inset 0 1px #ffffff29, 0 1px 2px 0 #fb414a3d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-disabled:inset-shadow-[0_1px_--theme(--color-white/16%)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #ffffff29, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #fb2c363d` / `inset 0 1px #ffffff29, 0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #fb414a3d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#fb2c36e6` / `#fb414ae6` | `hover:bg-destructive/90` |
| pressed | background-color | `#fb2c36e6` / `#fb414ae6` | `data-pressed:bg-destructive/90` |
| pressed | box-shadow | `inset 0 1px #00000014` | `[:disabled,:active,[data-pressed]]:shadow-none` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`> *[data-slot="button-loading-indicator"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#ffffff` | `color-white` | `*:data-[slot=button-loading-indicator]:text-white` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: variant=destructive-outline

```txt
border-input bg-popover not-dark:bg-clip-padding text-destructive-foreground shadow-xs/5 not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] hover:border-destructive/32 hover:bg-destructive/4 data-pressed:border-destructive/32 data-pressed:bg-destructive/4 *:data-[slot=button-loading-indicator]:text-foreground dark:bg-input/32 dark:not-disabled:before:shadow-[0_-1px_--theme(--color-white/2%)] dark:not-disabled:not-active:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [:disabled,:active,[data-pressed]]:shadow-none
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| background-color | `#ffffff` / `#ffffff07` | `popover` | `bg-popover` |
| color | `#c10007` / `#ff6467` | `destructive-foreground` | `text-destructive-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #0000000d` / `0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #0000000d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| pressed | border-color | `#fb2c3652` / `#fb414a52` | `data-pressed:border-destructive/32` |
| pressed | background-color | `#fb2c360a` / `#ffffff07` | `data-pressed:bg-destructive/4` |
| pressed | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| hover | border-color | `#fb2c3652` / `#fb414a52` | `hover:border-destructive/32` |
| hover | background-color | `#fb2c360a` / `#ffffff07` | `hover:bg-destructive/4` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | — / `0 -1px #ffffff05` | `dark:not-disabled:before:shadow-[0_-1px_--theme(--color-white/2%)]` |

**`> *[data-slot="button-loading-indicator"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626` / `#f5f5f5` | `foreground` | `*:data-[slot=button-loading-indicator]:text-foreground` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: variant=ghost

```txt
border-transparent text-foreground hover:bg-accent data-pressed:bg-accent *:data-[slot=button-loading-indicator]:text-foreground
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#00000000` | `transparent` | `border-transparent` |
| background-color | `(unset)` | `` | `(removed)` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `(unset)` | `` | `(removed)` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-accent` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `data-pressed:bg-accent` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`> *[data-slot="button-loading-indicator"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626` / `#f5f5f5` | `foreground` | `*:data-[slot=button-loading-indicator]:text-foreground` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: variant=link

```txt
border-transparent text-foreground underline-offset-4 hover:underline data-pressed:underline *:data-[slot=button-loading-indicator]:text-foreground
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#00000000` | `transparent` | `border-transparent` |
| background-color | `(unset)` | `` | `(removed)` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `(unset)` | `` | `(removed)` |
| text-underline-offset | `4px` | `4px` | `underline-offset-4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | text-decoration-line | `underline` | `hover:underline` |
| pressed | text-decoration-line | `underline` | `data-pressed:underline` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`> *[data-slot="button-loading-indicator"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626` / `#f5f5f5` | `foreground` | `*:data-[slot=button-loading-indicator]:text-foreground` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: variant=outline

```txt
border-input bg-popover not-dark:bg-clip-padding text-foreground shadow-xs/5 not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)] hover:bg-accent/50 data-pressed:bg-accent/50 *:data-[slot=button-loading-indicator]:text-foreground dark:bg-input/32 dark:data-pressed:bg-input/64 dark:hover:bg-input/64 dark:not-disabled:before:shadow-[0_-1px_--theme(--color-white/2%)] dark:not-disabled:not-active:not-data-pressed:before:shadow-[0_-1px_--theme(--color-white/6%)] [:disabled,:active,[data-pressed]]:shadow-none
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| background-color | `#ffffff` / `#ffffff07` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #0000000d` / `0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #0000000d` | `focus-visible:ring-2` |
| disabled | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| pressed | background-color | `#00000005` / `#ffffff0d` | `data-pressed:bg-accent/50` |
| pressed | box-shadow | `none` | `[:disabled,:active,[data-pressed]]:shadow-none` |
| hover | background-color | `#00000005` / `#ffffff0d` | `hover:bg-accent/50` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-disabled:not-active:not-data-pressed:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| pressed | box-shadow | — / `0 -1px #ffffff05` | `dark:not-disabled:before:shadow-[0_-1px_--theme(--color-white/2%)]` |

**`> *[data-slot="button-loading-indicator"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626` / `#f5f5f5` | `foreground` | `*:data-[slot=button-loading-indicator]:text-foreground` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### buttonVariants: variant=secondary

```txt
border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/90 data-pressed:bg-secondary/90 *:data-[slot=button-loading-indicator]:text-secondary-foreground [:active,[data-pressed]]:bg-secondary/80
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#00000000` | `transparent` | `border-transparent` |
| background-color | `#0000000a` / `#ffffff0a` | `secondary` | `bg-secondary` |
| color | `#262626` / `#f5f5f5` | `secondary-foreground` | `text-secondary-foreground` |
| box-shadow | `(unset)` | `` | `(removed)` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| [data-loading] | color | `#00000000` | `data-loading:text-transparent` |
| [data-loading] | -webkit-user-select | `none` | `data-loading:select-none` |
| [data-loading] | user-select | `none` | `data-loading:select-none` |
| hover | background-color | `#00000009` / `#ffffff09` | `hover:bg-secondary/90` |
| pressed | background-color | `#00000008` / `#ffffff08` | `[:active,[data-pressed]]:bg-secondary/80` |

**element — mobile differences**: height `36px` · font-size `16px` · line-height `24px`

**`::after` — mobile differences**: content `""` · position `absolute` · width `100%` · height `100%` · min-height `44px` · min-width `44px`

**`> *[data-slot="button-loading-indicator"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626` / `#f5f5f5` | `secondary-foreground` | `*:data-[slot=button-loading-indicator]:text-secondary-foreground` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `Button` → `[data-slot=button-loading-indicator]` · `<Spinner>` (line 76)

Prop defaults: `loading=false`

#### base

```txt
pointer-events-none absolute
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |

