# Badge

Source: `apps/ui/registry/default/ui/badge.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/badge.mdx`

> A badge or a component that looks like a badge.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `Badge`

## Behavior & keyboard

- Presentational `<span>`; when rendered as `<button>`/`<a>` (`render` prop) it gets hover styles via `[button&,a&]` selectors and normal button/link keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `badgeVariants` (line 9)

Defaults: {"size":"default","variant":"default"}

```txt
base: relative inline-flex shrink-0 items-center justify-center gap-1 whitespace-nowrap rounded-sm border border-transparent font-medium outline-none transition-shadow focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-3.5 sm:[&_svg:not([class*='size-'])]:size-3 [&_svg]:pointer-events-none [&_svg]:shrink-0 [button&,a&]:cursor-pointer [button&,a&]:pointer-coarse:after:absolute [button&,a&]:pointer-coarse:after:size-full [button&,a&]:pointer-coarse:after:min-h-11 [button&,a&]:pointer-coarse:after:min-w-11
size.default: h-5.5 min-w-5.5 px-[calc(--spacing(1)-1px)] text-sm sm:h-4.5 sm:min-w-4.5 sm:text-xs
size.lg: h-6.5 min-w-6.5 px-[calc(--spacing(1.5)-1px)] text-base sm:h-5.5 sm:min-w-5.5 sm:text-sm
size.sm: h-5 min-w-5 rounded-[.25rem] px-[calc(--spacing(1)-1px)] text-xs sm:h-4 sm:min-w-4 sm:text-[.625rem]
variant.default: bg-primary text-primary-foreground [button&,a&]:hover:bg-primary/90
variant.destructive: bg-destructive text-white [button&,a&]:hover:bg-destructive/90
variant.error: bg-destructive/8 text-destructive-foreground dark:bg-destructive/16
variant.info: bg-info/8 text-info-foreground dark:bg-info/16
variant.outline: border-input bg-background text-foreground dark:bg-input/32 [button&,a&]:hover:bg-accent/50 dark:[button&,a&]:hover:bg-input/48
variant.secondary: bg-secondary text-secondary-foreground [button&,a&]:hover:bg-secondary/90
variant.success: bg-success/8 text-success-foreground dark:bg-success/16
variant.warning: bg-warning/8 text-warning-foreground dark:bg-warning/16
```

## Parts

### `Badge` → `[data-slot=badge]` · `useRender <span>` (line 54)

#### base (cva defaults: size=default, variant=default)

```txt
relative inline-flex shrink-0 items-center justify-center gap-1 whitespace-nowrap rounded-sm border border-transparent font-medium outline-none transition-shadow focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-3.5 sm:[&_svg:not([class*='size-'])]:size-3 [&_svg]:pointer-events-none [&_svg]:shrink-0 [button&,a&]:cursor-pointer [button&,a&]:pointer-coarse:after:absolute [button&,a&]:pointer-coarse:after:size-full [button&,a&]:pointer-coarse:after:min-h-11 [button&,a&]:pointer-coarse:after:min-w-11 h-5.5 min-w-5.5 px-[calc(--spacing(1)-1px)] text-sm sm:h-4.5 sm:min-w-4.5 sm:text-xs bg-primary text-primary-foreground [button&,a&]:hover:bg-primary/90
```

**element — default**

Key metrics: display `inline-flex` · height `18px` · min-width `18px` · padding-inline `3px` · gap `4px` · border-width `1px` · border-color `#00000000` · border-radius `6px` · background-color `#262626` / `#f5f5f5` (primary) · color `#fafafa` / `#262626` (primary-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium) · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000000` | `transparent` | `border-transparent` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `18px` | `calc(var(--spacing) * 4.5)` | `sm:h-4.5` |
| min-width | `18px` | `calc(var(--spacing) * 4.5)` | `sm:min-w-4.5` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| background-color | `#262626` / `#f5f5f5` | `primary` | `bg-primary` |
| padding-inline | `3px` | `calc(calc(var(--spacing) * 1) - 1px)` | `px-[calc(--spacing(1)-1px)]` |
| font-size | `12px` | `text-xs` | `sm:text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `sm:text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| color | `#fafafa` / `#262626` | `primary-foreground` | `text-primary-foreground` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='opacity-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `[&_svg:not([class*='opacity-'])]:opacity-80` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `12px` | `calc(var(--spacing) * 3)` | `sm:[&_svg:not([class*='size-'])]:size-3` |
| height | `12px` | `calc(var(--spacing) * 3)` | `sm:[&_svg:not([class*='size-'])]:size-3` |

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: size=lg

```txt
h-6.5 min-w-6.5 px-[calc(--spacing(1.5)-1px)] text-base sm:h-5.5 sm:min-w-5.5 sm:text-sm
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `22px` | `calc(var(--spacing) * 5.5)` | `sm:h-5.5` |
| min-width | `22px` | `calc(var(--spacing) * 5.5)` | `sm:min-w-5.5` |
| padding-inline | `5px` | `calc(calc(var(--spacing) * 1.5) - 1px)` | `px-[calc(--spacing(1.5)-1px)]` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `26px` · min-width `26px` · font-size `16px` · line-height `24px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: size=sm

```txt
h-5 min-w-5 rounded-[.25rem] px-[calc(--spacing(1)-1px)] text-xs sm:h-4 sm:min-w-4 sm:text-[.625rem]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:h-4` |
| min-width | `16px` | `calc(var(--spacing) * 4)` | `sm:min-w-4` |
| font-size | `10px` | `.625rem` | `sm:text-[.625rem]` |
| line-height | `13.333px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `20px` · min-width `20px` · font-size `12px` · line-height `16px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=destructive

```txt
bg-destructive text-white [button&,a&]:hover:bg-destructive/90
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#fb2c36` / `#fb414a` | `destructive` | `bg-destructive` |
| color | `#ffffff` | `color-white` | `text-white` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=error

```txt
bg-destructive/8 text-destructive-foreground dark:bg-destructive/16
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#fb2c3614` / `#fb414a29` | `destructive/8%` | `bg-destructive/8` |
| color | `#c10007` / `#ff6467` | `destructive-foreground` | `text-destructive-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=info

```txt
bg-info/8 text-info-foreground dark:bg-info/16
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#2b7fff14` / `#2b7fff29` | `info/8%` | `bg-info/8` |
| color | `#1447e6` / `#51a2ff` | `info-foreground` | `text-info-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=outline

```txt
border-input bg-background text-foreground dark:bg-input/32 [button&,a&]:hover:bg-accent/50 dark:[button&,a&]:hover:bg-input/48
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=secondary

```txt
bg-secondary text-secondary-foreground [button&,a&]:hover:bg-secondary/90
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#0000000a` / `#ffffff0a` | `secondary` | `bg-secondary` |
| color | `#262626` / `#f5f5f5` | `secondary-foreground` | `text-secondary-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=success

```txt
bg-success/8 text-success-foreground dark:bg-success/16
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#00bc7d14` / `#00bc7d29` | `success/8%` | `bg-success/8` |
| color | `#007a55` / `#00d492` | `success-foreground` | `text-success-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

#### badgeVariants: variant=warning

```txt
bg-warning/8 text-warning-foreground dark:bg-warning/16
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#fe9a0014` / `#fe9a0029` | `warning/8%` | `bg-warning/8` |
| color | `#bb4d00` / `#ffb900` | `warning-foreground` | `text-warning-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |
| tag(button) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |
| tag(a) | cursor | `pointer` | `[button&,a&]:cursor-pointer` |

**element — mobile differences**: height `22px` · min-width `22px` · font-size `14px` · line-height `20px`

**`svg:not([class*='size-'])` — mobile differences**: width `14px` · height `14px`

