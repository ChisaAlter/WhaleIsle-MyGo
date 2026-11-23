# chat-conversation

Source: `apps/ui/registry/default/chat/chat-conversation.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `@/ui/scroll-area`

Exports: `ChatConversation`

## Behavior & keyboard

- Scroll container managing stick-to-bottom, jump-to-latest, and unread divider (hooks `use-chat-scroll`, `use-chat-popup-dismissal`).

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatConversation` → `[data-slot=chat-conversation]` · `<div>` (line 36)

Prop defaults: `inactive=false`

#### base

```txt
relative isolate flex min-h-0 flex-1 flex-col
```

**element — default**

Key metrics: display `flex` · min-height `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| isolation | `isolate` | `isolate` | `isolate` |
| display | `flex` | `flex` | `flex` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| flex | `1` | `1` | `flex-1` |
| flex-direction | `column` | `column` | `flex-col` |

### `ChatConversation` → (no data-slot) · `<ScrollAreaPrimitive.Root>` (line 41)

Attributes: `inert={inactive}`

Prop defaults: `inactive=false`

#### base

```txt
size-full min-h-0 touch-auto before:pointer-events-none before:absolute before:inset-x-0 before:top-0 before:z-10 before:h-6 before:bg-linear-to-b before:from-popover before:to-transparent before:opacity-0 before:content-[''] after:pointer-events-none after:absolute after:inset-x-0 after:bottom-0 after:z-10 after:h-6 after:bg-linear-to-t after:from-popover after:to-transparent after:opacity-0 after:content-[''] data-overflow-y-end:after:opacity-100 data-overflow-y-start:before:opacity-100
```

**element — default**

Key metrics: height `100%` · min-height `0px` · width `100%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| touch-action | `auto` | `auto` | `touch-auto` |

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `''` | `var(--tw-content)` | `after:content-['']` |
| pointer-events | `none` | `none` | `after:pointer-events-none` |
| position | `absolute` | `absolute` | `after:absolute` |
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `after:inset-x-0` |
| bottom | `0px` | `calc(var(--spacing) * 0)` | `after:bottom-0` |
| z-index | `10` | `10` | `after:z-10` |
| height | `24px` | `calc(var(--spacing) * 6)` | `after:h-6` |
| background-image | `linear-gradient(to top in oklab, #fff 0%, transparent 100%)` / `linear-gradient(to top in oklab, #1d1d1d 0%, transparent 100%)` | `linear-gradient(var(--tw-gradient-stops))` | `after:bg-linear-to-t` |
| opacity | `0%` | `0%` | `after:opacity-0` |

**`::after` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-overflow-y-end] | opacity | `100%` | `data-overflow-y-end:after:opacity-100` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `''` | `var(--tw-content)` | `before:content-['']` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `before:inset-x-0` |
| top | `0px` | `calc(var(--spacing) * 0)` | `before:top-0` |
| z-index | `10` | `10` | `before:z-10` |
| height | `24px` | `calc(var(--spacing) * 6)` | `before:h-6` |
| background-image | `linear-gradient(to bottom in oklab, #fff 0%, transparent 100%)` / `linear-gradient(to bottom in oklab, #1d1d1d 0%, transparent 100%)` | `linear-gradient(var(--tw-gradient-stops))` | `before:bg-linear-to-b` |
| opacity | `0%` | `0%` | `before:opacity-0` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-overflow-y-start] | opacity | `100%` | `data-overflow-y-start:before:opacity-100` |

### `ChatConversation` → `[data-slot=scroll-area-viewport]` · `<ScrollAreaPrimitive.Viewport>` (line 45)

Attributes: `role=log`, `aria-label={label}`, `aria-live={live}`, `aria-relevant=additions text`

Prop defaults: `inactive=false`

#### base

```txt
h-full rounded-[inherit] outline-none transition-shadow focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset data-has-overflow-y:overscroll-y-contain data-has-overflow-x:overscroll-x-contain
```

**element — default**

Key metrics: height `100%` · border-radius `inherit` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `100%` | `100%` | `h-full` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 0 0 2px #a1a1a1` / `inset 0 0 0 2px #737373` | `focus-visible:ring-2` |
| [data-has-overflow-y] | overscroll-behavior-y | `contain` | `data-has-overflow-y:overscroll-y-contain` |
| [data-has-overflow-x] | overscroll-behavior-x | `contain` | `data-has-overflow-x:overscroll-x-contain` |

### `ChatConversation` → `[data-slot=scroll-area-content]` · `<ScrollAreaPrimitive.Content>` (line 55)

Attributes: `style={{ minWidth: 0 }}`

Prop defaults: `inactive=false`

Dynamic className inputs (not statically resolvable): `contentClassName`

#### base

```txt
px-4 pb-4
```

**element — default**

Key metrics: padding-inline `16px` · padding-bottom `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `16px` | `calc(var(--spacing) * 4)` | `px-4` |
| padding-bottom | `16px` | `calc(var(--spacing) * 4)` | `pb-4` |

### `ChatConversation` → `[data-slot=scroll-area-corner]` · `<ScrollAreaPrimitive.Corner>` (line 65)

Prop defaults: `inactive=false`

### `ChatConversation` → (no data-slot) · `<div>` (line 68)

Prop defaults: `inactive=false`

#### base

```txt
pointer-events-none absolute inset-x-0 bottom-3 z-30 flex justify-center
```

**element — default**

Key metrics: display `flex` · z-index `30`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| inset-inline | `0px` | `calc(var(--spacing) * 0)` | `inset-x-0` |
| bottom | `12px` | `calc(var(--spacing) * 3)` | `bottom-3` |
| z-index | `30` | `30` | `z-30` |
| display | `flex` | `flex` | `flex` |
| justify-content | `center` | `center` | `justify-center` |

