# Empty

Source: `apps/ui/registry/default/ui/empty.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/empty.mdx`

> A container for displaying empty state information.

Depends on: —

Exports: `Empty`, `EmptyHeader`, `EmptyMedia`, `EmptyTitle`, `EmptyDescription`, `EmptyContent`

## Behavior & keyboard

- Presentational empty-state block; no keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `emptyMediaVariants` (line 5)

Defaults: {"variant":"default"}

```txt
base: flex shrink-0 items-center justify-center [&_svg]:pointer-events-none [&_svg]:shrink-0
variant.default: bg-transparent
variant.icon: relative flex size-9 shrink-0 items-center justify-center rounded-md border bg-card not-dark:bg-clip-padding text-foreground shadow-sm/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-md)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='size-'])]:size-4.5
```

## Parts

### `Empty` → `[data-slot=empty]` · `<div>` (line 25)

#### base

```txt
flex min-w-0 flex-1 flex-col items-center justify-center gap-6 text-balance px-6 py-12 text-center md:py-20
```

**element — default**

Key metrics: display `flex` · min-width `0px` · padding-inline `24px` · padding-block `80px` · gap `24px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex | `1` | `1` | `flex-1` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `24px` | `calc(var(--spacing) * 6)` | `gap-6` |
| padding-inline | `24px` | `calc(var(--spacing) * 6)` | `px-6` |
| padding-block | `80px` | `calc(var(--spacing) * 20)` | `md:py-20` |
| text-align | `center` | `center` | `text-center` |
| text-wrap | `balance` | `balance` | `text-balance` |

**element — mobile differences**: padding-block `48px`

### `EmptyHeader` → `[data-slot=empty-header]` · `<div>` (line 41)

#### base

```txt
flex max-w-sm flex-col items-center text-center
```

**element — default**

Key metrics: display `flex` · max-width `384px` (container-sm)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| max-width | `384px` | `container-sm` | `max-w-sm` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `center` | `center` | `items-center` |
| text-align | `center` | `center` | `text-center` |

### `EmptyMedia` → `[data-slot=empty-media]` · `<div>` (line 59)

Attributes: `data-variant={variant}`

Prop defaults: `variant="default"`

#### base

```txt
relative mb-6
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| margin-bottom | `24px` | `calc(var(--spacing) * 6)` | `mb-6` |

### `EmptyMedia` → (no data-slot) · `<div>` (line 67)

Attributes: `aria-hidden=true`

Prop defaults: `variant="default"`

#### base (cva defaults: variant=default)

```txt
flex shrink-0 items-center justify-center [&_svg]:pointer-events-none [&_svg]:shrink-0 bg-transparent pointer-events-none absolute bottom-px origin-bottom-left -translate-x-0.5 -rotate-10 scale-84 shadow-none
```

**element — default**

Key metrics: display `flex` · background-color `#00000000` · box-shadow `none`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| bottom | `1px` | `1px` | `bottom-px` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| transform-origin | `0 100%` | `0 100%` | `origin-bottom-left` |
| translate | `-2px 0` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-x-0.5` |
| scale | `84% 84%` | `var(--tw-scale-x) var(--tw-scale-y)` | `scale-84` |
| rotate | `-10deg` | `calc(10deg * -1)` | `-rotate-10` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |
| box-shadow | `none` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-none` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

#### emptyMediaVariants: variant=icon

```txt
relative flex size-9 shrink-0 items-center justify-center rounded-md border bg-card not-dark:bg-clip-padding text-foreground shadow-sm/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-md)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='size-'])]:size-4.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| background-color | `#ffffff` / `#191919` | `card` | `bg-card` |
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| width | `36px` | `calc(var(--spacing) * 9)` | `size-9` |
| height | `36px` | `calc(var(--spacing) * 9)` | `size-9` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`svg:not([class*='size-'])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `18px` | `calc(var(--spacing) * 4.5)` | `[&_svg:not([class*='size-'])]:size-4.5` |
| height | `18px` | `calc(var(--spacing) * 4.5)` | `[&_svg:not([class*='size-'])]:size-4.5` |

### `EmptyMedia` → (no data-slot) · `<div>` (line 74)

Attributes: `aria-hidden=true`

Prop defaults: `variant="default"`

#### base (cva defaults: variant=default)

```txt
flex shrink-0 items-center justify-center [&_svg]:pointer-events-none [&_svg]:shrink-0 bg-transparent pointer-events-none absolute bottom-px origin-bottom-right translate-x-0.5 rotate-10 scale-84 shadow-none
```

**element — default**

Key metrics: display `flex` · background-color `#00000000` · box-shadow `none`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| bottom | `1px` | `1px` | `bottom-px` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| transform-origin | `100% 100%` | `100% 100%` | `origin-bottom-right` |
| translate | `2px 0` | `var(--tw-translate-x) var(--tw-translate-y)` | `translate-x-0.5` |
| scale | `84% 84%` | `var(--tw-scale-x) var(--tw-scale-y)` | `scale-84` |
| rotate | `10deg` | `10deg` | `rotate-10` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |
| box-shadow | `none` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-none` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

#### emptyMediaVariants: variant=icon

```txt
relative flex size-9 shrink-0 items-center justify-center rounded-md border bg-card not-dark:bg-clip-padding text-foreground shadow-sm/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-md)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='size-'])]:size-4.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| background-color | `#ffffff` / `#191919` | `card` | `bg-card` |
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| width | `36px` | `calc(var(--spacing) * 9)` | `size-9` |
| height | `36px` | `calc(var(--spacing) * 9)` | `size-9` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`svg:not([class*='size-'])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `18px` | `calc(var(--spacing) * 4.5)` | `[&_svg:not([class*='size-'])]:size-4.5` |
| height | `18px` | `calc(var(--spacing) * 4.5)` | `[&_svg:not([class*='size-'])]:size-4.5` |

### `EmptyMedia` → (no data-slot) · `<div>` (line 83)

Prop defaults: `variant="default"`

#### base (cva defaults: variant=default)

```txt
flex shrink-0 items-center justify-center [&_svg]:pointer-events-none [&_svg]:shrink-0 bg-transparent
```

**element — default**

Key metrics: display `flex` · background-color `#00000000`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

#### emptyMediaVariants: variant=icon

```txt
relative flex size-9 shrink-0 items-center justify-center rounded-md border bg-card not-dark:bg-clip-padding text-foreground shadow-sm/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-md)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='size-'])]:size-4.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#ffffff` / `#191919` | `card` | `bg-card` |
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| width | `36px` | `calc(var(--spacing) * 9)` | `size-9` |
| height | `36px` | `calc(var(--spacing) * 9)` | `size-9` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-sm/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`svg:not([class*='size-'])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `18px` | `calc(var(--spacing) * 4.5)` | `[&_svg:not([class*='size-'])]:size-4.5` |
| height | `18px` | `calc(var(--spacing) * 4.5)` | `[&_svg:not([class*='size-'])]:size-4.5` |

### `EmptyTitle` → `[data-slot=empty-title]` · `<div>` (line 96)

#### base

```txt
font-bold font-heading text-xl
```

**element — default**

Key metrics: font-size `20px` (text-xl) · line-height `28px` · font-weight `700` (font-weight-bold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-family | `ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` | `font-sans` | `font-heading` |
| font-size | `20px` | `text-xl` | `text-xl` |
| line-height | `28px` | `var(--tw-leading, var(--text-xl--line-height))` | `text-xl` |
| font-weight | `700` | `font-weight-bold` | `font-bold` |

### `EmptyDescription` → `[data-slot=empty-description]` · `<div>` (line 109)

#### base

```txt
text-muted-foreground text-sm [&>a:hover]:text-primary [&>a]:underline [&>a]:underline-offset-4 [[data-slot=empty-title]+&]:mt-1
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| prev([data-slot=empty-title]) | margin-top | `4px` | `[[data-slot=empty-title]+&]:mt-1` |

**`>a` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| text-decoration-line | `underline` | `underline` | `[&>a]:underline` |
| text-underline-offset | `4px` | `4px` | `[&>a]:underline-offset-4` |

**`>a:hover` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626` / `#f5f5f5` | `primary` | `[&>a:hover]:text-primary` |

### `EmptyContent` → `[data-slot=empty-content]` · `<div>` (line 125)

#### base

```txt
flex w-full min-w-0 max-w-sm flex-col items-center gap-4 text-balance text-sm
```

**element — default**

Key metrics: display `flex` · width `100%` · min-width `0px` · max-width `384px` (container-sm) · gap `16px` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| max-width | `384px` | `container-sm` | `max-w-sm` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `center` | `center` | `items-center` |
| gap | `16px` | `calc(var(--spacing) * 4)` | `gap-4` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| text-wrap | `balance` | `balance` | `text-balance` |

