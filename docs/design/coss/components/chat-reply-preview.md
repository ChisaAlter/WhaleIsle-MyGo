# chat-reply-preview

Source: `apps/ui/registry/default/chat/chat-reply-preview.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`, `@/ui/button`, `@/ui/scroll-area`, `@/ui/tooltip`

Exports: `ChatReplyPreview`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatReplyPreview` → (no data-slot) · `<motion.button>` (line 67)

Attributes: `type=button`, `tabIndex={-1}`, `aria-label={cancelLabel}`, `initial={{ opacity: 0 }}`, `animate={{ opacity: isPresent ? 1 : 0 }}`, `transition={{ duration: reduceMotion ? 0 : 0.22 }}`

#### base

```txt
absolute inset-0 z-40 cursor-default touch-none bg-popover/32 backdrop-blur-sm before:pointer-events-none before:absolute before:inset-x-0 before:top-0 before:h-6 before:bg-linear-to-b before:from-popover before:to-transparent before:content-['']
```

**element — default**

Key metrics: background-color `#ffffff52` / `#1d1d1d52` (popover/32%) · z-index `40`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `40` | `40` | `z-40` |
| cursor | `default` | `default` | `cursor-default` |
| touch-action | `none` | `none` | `touch-none` |
| background-color | `#ffffff52` / `#1d1d1d52` | `popover/32%` | `bg-popover/32` |
| -webkit-backdrop-filter | `blur(8px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-sm` |
| backdrop-filter | `blur(8px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-sm` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `''` | `var(--tw-content)` | `before:content-['']` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `before:inset-x-0` |
| top | `0px` | `calc(var(--spacing) * 0)` | `before:top-0` |
| height | `24px` | `calc(var(--spacing) * 6)` | `before:h-6` |
| background-image | `linear-gradient(to bottom in oklab, #fff 0%, transparent 100%)` / `linear-gradient(to bottom in oklab, #1d1d1d 0%, transparent 100%)` | `linear-gradient(var(--tw-gradient-stops))` | `before:bg-linear-to-b` |

### `ChatReplyPreview` → (no data-slot) · `<div>` (line 81)

#### base

```txt
pointer-events-none absolute inset-0 z-50 overflow-clip
```

**element — default**

Key metrics: z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| overflow | `clip` | `clip` | `overflow-clip` |

### `ChatReplyPreview` → (no data-slot) · `<ScrollArea>` (line 82)

Attributes: `overscrollContain=true`

#### base

```txt
**:data-[slot=scroll-area-content]:flex **:data-[slot=scroll-area-content]:min-h-full **:data-[slot=scroll-area-content]:flex-col **:data-[slot=scroll-area-content]:py-4 [&_[data-slot=scroll-area-scrollbar]]:pointer-events-auto
```

**`[data-slot=scroll-area-scrollbar]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `auto` | `auto` | `[&_[data-slot=scroll-area-scrollbar]]:pointer-events-auto` |

**`*[data-slot="scroll-area-content"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `**:data-[slot=scroll-area-content]:flex` |
| min-height | `100%` | `100%` | `**:data-[slot=scroll-area-content]:min-h-full` |
| flex-direction | `column` | `column` | `**:data-[slot=scroll-area-content]:flex-col` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `**:data-[slot=scroll-area-content]:py-4` |

### `ChatReplyPreview` → `[data-slot=chat-reply-focus]` · `<div>` (line 86)

#### base

```txt
relative mx-4 mt-auto shrink-0
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| margin-inline | `16px` | `calc(var(--spacing) * 4)` | `mx-4` |
| margin-top | `auto` | `auto` | `mt-auto` |
| flex-shrink | `0` | `0` | `shrink-0` |

### `ChatReplyPreview` → (no data-slot) · `<motion.div>` (line 91)

Attributes: `aria-hidden=true`, `initial={{ opacity: 0 }}`, `animate={{ opacity: isPresent ? 1 : 0 }}`, `transition={{ duration: reduceMotion ? 0 : 0.22 }}`

#### base

```txt
absolute -inset-x-4 -top-20 -bottom-4 bg-linear-to-t from-[calc(100%-4rem)] from-popover to-transparent
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset-inline | `-16px` | `calc(var(--spacing) * -4)` | `-inset-x-4` |
| top | `-80px` | `calc(var(--spacing) * -20)` | `-top-20` |
| bottom | `-16px` | `calc(var(--spacing) * -4)` | `-bottom-4` |
| background-image | `linear-gradient(to top in oklab, #fff calc(100% - 64px), transparent 100%)` / `linear-gradient(to top in oklab, #1d1d1d calc(100% - 64px), transparent 100%)` | `linear-gradient(var(--tw-gradient-stops))` | `bg-linear-to-t` |

### `ChatReplyPreview` → (no data-slot) · `<motion.div>` (line 98)

Attributes: `initial={reduceMotion ? false : { transform: `translateY(${origin ?? 0}px)`, opacity: 1 }}`, `animate={isPresent ? { transform: "translateY(0px)", opacity: 1 } : { transform: returning ? `translateY(${origin ?? 0}px)` : "t`, `transition={reduceMotion ? { duration: 0 } : isPresent \|\| returning ? { type: "tween", duration: 0.35, ease: [0.3, 0.84, 0.24, 1]`

#### base

```txt
relative grid items-start gap-y-1
```

**element — default**

Key metrics: display `grid` · row-gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `grid` | `grid` | `grid` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| row-gap | `4px` | `calc(var(--spacing) * 1)` | `gap-y-1` |

#### when `own`

```txt
grid-cols-1
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-template-columns | `repeat(1, minmax(0, 1fr))` | `repeat(1, minmax(0, 1fr))` | `grid-cols-1` |

#### when `!(own)`

```txt
grid-cols-[1.75rem_minmax(0,1fr)] gap-x-2
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-template-columns | `28px minmax(0,1fr)` | `1.75rem minmax(0,1fr)` | `grid-cols-[1.75rem_minmax(0,1fr)]` |
| column-gap | `8px` | `calc(var(--spacing) * 2)` | `gap-x-2` |

#### when `origin === null`

```txt
invisible
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| visibility | `hidden` | `hidden` | `invisible` |

### `ChatReplyPreview` → (no data-slot) · `<motion.div>` (line 138)

Attributes: `initial={{ opacity: target.firstInGroup ? 1 : 0 }}`, `animate={{ opacity: isPresent \|\| target.firstInGroup ? 1 : 0, }}`, `transition={{ duration: reduceMotion ? 0 : 0.18 }}`

#### base

```txt
col-start-1 row-start-2 w-7
```

**element — default**

Key metrics: width `28px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| grid-row-start | `2` | `2` | `row-start-2` |
| width | `28px` | `calc(var(--spacing) * 7)` | `w-7` |

### `ChatReplyPreview` → (no data-slot) · `<motion.div>` (line 149)

Attributes: `initial={{ opacity: !own && target.firstInGroup ? 1 : 0 }}`, `animate={{ opacity: isPresent \|\| (!own && target.firstInGroup) ? 1 : 0, }}`, `transition={{ duration: reduceMotion ? 0 : 0.18 }}`

#### base

```txt
flex h-lh min-w-0 max-w-full items-center gap-2 px-1 font-medium text-muted-foreground text-xs
```

**element — default**

Key metrics: display `flex` · height `1lh` · min-width `0px` · max-width `100%` · padding-inline `4px` · gap `8px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `1lh` | `1lh` | `h-lh` |
| max-width | `100%` | `100%` | `max-w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding-inline | `4px` | `calc(var(--spacing) * 1)` | `px-1` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

#### when `own`

```txt
justify-self-end
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| justify-self | `flex-end` | `flex-end` | `justify-self-end` |

#### when `!(own)`

```txt
col-start-2 row-start-1
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| grid-row-start | `1` | `1` | `row-start-1` |

### `ChatReplyPreview` → (no data-slot) · `<span>` (line 160)

Attributes: `role=status`

#### base

```txt
min-w-0 truncate
```

**element — default**

Key metrics: min-width `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| overflow | `hidden` | `hidden` | `truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `truncate` |
| white-space | `nowrap` | `nowrap` | `truncate` |

### `ChatReplyPreview` → (no data-slot) · `<div>` (line 164)

Attributes: `style={{ width: target.width }}`

#### base

```txt
pointer-events-auto relative row-start-2 max-w-full rounded-2xl bg-popover
```

**element — default**

Key metrics: max-width `100%` · border-radius `16px` (radius-2xl) · background-color `#ffffff` / `#1d1d1d` (popover)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `auto` | `auto` | `pointer-events-auto` |
| position | `relative` | `relative` | `relative` |
| grid-row-start | `2` | `2` | `row-start-2` |
| max-width | `100%` | `100%` | `max-w-full` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |

#### when `own`

```txt
justify-self-end
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| justify-self | `flex-end` | `flex-end` | `justify-self-end` |

#### when `!(own)`

```txt
col-start-2
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |

#### when `target.firstInGroup && own`

```txt
rounded-se-md
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-se-md` |

#### when `target.firstInGroup && !(own)`

```txt
rounded-ss-md
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-ss-md` |

### `ChatReplyPreview` → (no data-slot) · `<motion.div>` (line 180)

Attributes: `initial={{ opacity: 0 }}`, `animate={{ opacity: isPresent ? 1 : 0 }}`, `transition={{ duration: reduceMotion ? 0 : 0.18 }}`

#### base

```txt
absolute -top-2.5 z-10 flex rounded-full bg-popover
```

**element — default**

Key metrics: display `flex` · border-radius `9999px` · background-color `#ffffff` / `#1d1d1d` (popover) · z-index `10`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| top | `-10px` | `calc(var(--spacing) * -2.5)` | `-top-2.5` |
| z-index | `10` | `10` | `z-10` |
| display | `flex` | `flex` | `flex` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |

#### when `own`

```txt
-start-2.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-start | `-10px` | `calc(var(--spacing) * -2.5)` | `-start-2.5` |

#### when `!(own)`

```txt
-end-2.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-end | `-10px` | `calc(var(--spacing) * -2.5)` | `-end-2.5` |

### `ChatReplyPreview` → (no data-slot) · `<Button>` (line 192)

Attributes: `type=button`, `size=icon-xs`, `variant=outline`, `aria-label={cancelLabel}`

#### base

```txt
size-6 rounded-full before:rounded-full
```

**element — default**

Key metrics: height `24px` · width `24px` · border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `24px` | `calc(var(--spacing) * 6)` | `size-6` |
| height | `24px` | `calc(var(--spacing) * 6)` | `size-6` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

