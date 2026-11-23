# chat-states

Source: `apps/ui/registry/default/chat/chat-states.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `@/ui/empty`, `@/ui/skeleton`

Exports: `ChatEmptyState`, `ChatLoadingState`, `ChatStatusBar`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatEmptyState` → (no data-slot) · `<Empty>` (line 36)

Attributes: `role=status`

#### base

```txt
min-h-64 gap-4 py-10 md:py-10
```

**element — default**

Key metrics: min-height `256px` · padding-block `40px` · gap `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `256px` | `calc(var(--spacing) * 64)` | `min-h-64` |
| gap | `16px` | `calc(var(--spacing) * 4)` | `gap-4` |
| padding-block | `40px` | `calc(var(--spacing) * 10)` | `md:py-10` |

### `ChatEmptyState` → (no data-slot) · `<EmptyMedia>` (line 38)

Attributes: `variant=icon`

#### base

```txt
mb-2 bg-popover
```

**element — default**

Key metrics: background-color `#ffffff` / `#1d1d1d` (popover)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-bottom | `8px` | `calc(var(--spacing) * 2)` | `mb-2` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |

### `ChatLoadingState` → (no data-slot) · `<div>` (line 66)

Attributes: `role=status`

#### base

```txt
flex flex-col gap-5 py-4
```

**element — default**

Key metrics: display `flex` · padding-block `16px` · gap `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `20px` | `calc(var(--spacing) * 5)` | `gap-5` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

### `ChatLoadingState` → (no data-slot) · `<span>` (line 67)

#### base

```txt
sr-only
```

**element — default**

Key metrics: height `1px` · width `1px` · padding `0` · border-width `0` · border-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `0` | `0` | `sr-only` |
| border-style | `solid` | `solid` | `(preflight) *` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `absolute` | `absolute` | `sr-only` |
| width | `1px` | `1px` | `sr-only` |
| height | `1px` | `1px` | `sr-only` |
| padding | `0` | `0` | `sr-only` |
| margin | `-1px` | `-1px` | `sr-only` |
| overflow | `hidden` | `hidden` | `sr-only` |
| clip-path | `inset(50%)` | `inset(50%)` | `sr-only` |
| white-space | `nowrap` | `nowrap` | `sr-only` |

### `ChatLoadingState` → (no data-slot) · `<Skeleton>` (line 77)

#### base

```txt
h-3 w-20 motion-reduce:animate-none
```

**element — default**

Key metrics: height `12px` · width `80px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `12px` | `calc(var(--spacing) * 3)` | `h-3` |
| width | `80px` | `calc(var(--spacing) * 20)` | `w-20` |

### `ChatLoadingState` → (no data-slot) · `<Skeleton>` (line 80)

#### base

```txt
size-7 rounded-full motion-reduce:animate-none
```

**element — default**

Key metrics: height `28px` · width `28px` · border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `28px` | `calc(var(--spacing) * 7)` | `size-7` |
| height | `28px` | `calc(var(--spacing) * 7)` | `size-7` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

### `ChatLoadingState` → (no data-slot) · `<Skeleton>` (line 85)

#### base

```txt
h-11 w-56 max-w-[83%] rounded-2xl motion-reduce:animate-none
```

**element — default**

Key metrics: height `44px` · width `224px` · max-width `83%` · border-radius `16px` (radius-2xl)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `44px` | `calc(var(--spacing) * 11)` | `h-11` |
| width | `224px` | `calc(var(--spacing) * 56)` | `w-56` |
| max-width | `83%` | `83%` | `max-w-[83%]` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |

#### when `side === "incoming"`

```txt
rounded-ss-md
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-ss-md` |

#### when `!(side === "incoming")`

```txt
rounded-se-md
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-se-md` |

### `ChatLoadingState` → (no data-slot) · `<Skeleton>` (line 91)

#### base

```txt
mx-1 h-2 w-9 motion-reduce:animate-none
```

**element — default**

Key metrics: height `8px` · width `36px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `4px` | `calc(var(--spacing) * 1)` | `mx-1` |
| height | `8px` | `calc(var(--spacing) * 2)` | `h-2` |
| width | `36px` | `calc(var(--spacing) * 9)` | `w-9` |

### `ChatStatusBar` → (no data-slot) · `<div>` (line 110)

Attributes: `role=status`

#### base

```txt
flex shrink-0 items-center justify-between gap-3 border-border/60 border-y bg-muted/72 px-5 py-2.5
```

**element — default**

Key metrics: display `flex` · padding-inline `20px` · padding-block `10px` · gap `12px` · border-color `#0000000c` / `#ffffff09` (border/60%) · background-color `#00000007` / `#ffffff07` (muted/72%)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#0000000c` / `#ffffff09` | `border/60%` | `border-border/60` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `12px` | `calc(var(--spacing) * 3)` | `gap-3` |
| border-block-style | `solid` | `var(--tw-border-style)` | `border-y` |
| border-block-width | `1px` | `1px` | `border-y` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `bg-muted/72` |
| padding-inline | `20px` | `calc(var(--spacing) * 5)` | `px-5` |
| padding-block | `10px` | `calc(var(--spacing) * 2.5)` | `py-2.5` |

### `ChatStatusBar` → (no data-slot) · `<div>` (line 114)

#### base

```txt
text-xs
```

**element — default**

Key metrics: font-size `12px` (text-xs) · line-height `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |

### `ChatStatusBar` → (no data-slot) · `<p>` (line 115)

#### base

```txt
m-0 font-medium
```

**element — default**

Key metrics: font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `0px` | `calc(var(--spacing) * 0)` | `m-0` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |

### `ChatStatusBar` → (no data-slot) · `<p>` (line 116)

#### base

```txt
m-0 mt-1 text-muted-foreground
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `0px` | `calc(var(--spacing) * 0)` | `m-0` |
| margin-top | `4px` | `calc(var(--spacing) * 1)` | `mt-1` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

