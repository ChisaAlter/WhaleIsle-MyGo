# chat-message-layout

Source: `apps/ui/registry/default/chat/chat-message-layout.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `ChatMessageLayout`, `ChatMessageAvatar`, `ChatMessageHeader`, `ChatMessageContent`, `ChatMessageFooter`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMessageLayout` → `[data-slot=chat-message]` · `useRender <article>` (line 18)

Attributes: `data-align={align}`

Prop defaults: `align="start"`

#### base

```txt
group/message relative grid scroll-m-8 grid-cols-1 items-start rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring has-[>[data-slot=chat-message-avatar]]:grid-cols-[auto_minmax(0,1fr)] has-[>[data-slot=chat-message-avatar]]:gap-x-2 has-[>[data-slot=chat-message-header]]:gap-y-1
```

Classes that produce no CSS (custom markers / group names): `group/message`

**element — default**

Key metrics: display `grid` · border-radius `14px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `grid` | `grid` | `grid` |
| scroll-margin | `32px` | `calc(var(--spacing) * 8)` | `scroll-m-8` |
| grid-template-columns | `repeat(1, minmax(0, 1fr))` | `repeat(1, minmax(0, 1fr))` | `grid-cols-1` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| border-radius | `14px` | `calc(var(--radius) + 4px)` | `rounded-xl` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` / `0 0 0 2px #737373` | `focus-visible:ring-2` |
| has(>[data-slot=chat-message-avatar]) | grid-template-columns | `auto minmax(0,1fr)` | `has-[>[data-slot=chat-message-avatar]]:grid-cols-[auto_minmax(0,1fr)]` |
| has(>[data-slot=chat-message-avatar]) | column-gap | `8px` | `has-[>[data-slot=chat-message-avatar]]:gap-x-2` |
| has(>[data-slot=chat-message-header]) | row-gap | `4px` | `has-[>[data-slot=chat-message-header]]:gap-y-1` |

### `ChatMessageAvatar` → `[data-slot=chat-message-avatar]` · `<div>` (line 39)

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

### `ChatMessageHeader` → `[data-slot=chat-message-header]` · `<div>` (line 52)

#### base

```txt
col-start-1 in-[[data-slot=chat-message]:has(>[data-slot=chat-message-avatar])]:col-start-2 row-start-1 px-1 font-medium text-muted-foreground text-xs group-data-[align=end]/message:justify-self-end
```

**element — default**

Key metrics: padding-inline `4px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| grid-row-start | `1` | `1` | `row-start-1` |
| padding-inline | `4px` | `calc(var(--spacing) * 1)` | `px-1` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=chat-message]:has(>[data-slot=chat-message-avatar])))) | grid-column-start | `2` | `in-[[data-slot=chat-message]:has(>[data-slot=chat-message-avatar])]:col-start-2` |
| in(.group\/message[data-align=end]) | justify-self | `flex-end` | `group-data-[align=end]/message:justify-self-end` |

### `ChatMessageContent` → `[data-slot=chat-message-content]` · `<div>` (line 68)

#### base

```txt
col-start-1 in-[[data-slot=chat-message]:has(>[data-slot=chat-message-avatar])]:col-start-2 row-start-2 flex min-w-0 flex-col gap-1 group-data-[align=end]/message:items-end
```

**element — default**

Key metrics: display `flex` · min-width `0px` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| grid-row-start | `2` | `2` | `row-start-2` |
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=chat-message]:has(>[data-slot=chat-message-avatar])))) | grid-column-start | `2` | `in-[[data-slot=chat-message]:has(>[data-slot=chat-message-avatar])]:col-start-2` |
| in(.group\/message[data-align=end]) | align-items | `flex-end` | `group-data-[align=end]/message:items-end` |

### `ChatMessageFooter` → `[data-slot=chat-message-footer]` · `<div>` (line 84)

#### base

```txt
flex items-center gap-1.5 px-1 pt-0.5 text-[.625rem] text-muted-foreground
```

**element — default**

Key metrics: display `flex` · padding-inline `4px` · padding-top `2px` · gap `6px` · color `#686868` / `#818181` (muted-foreground) · font-size `10px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |
| padding-inline | `4px` | `calc(var(--spacing) * 1)` | `px-1` |
| padding-top | `2px` | `calc(var(--spacing) * 0.5)` | `pt-0.5` |
| font-size | `10px` | `.625rem` | `text-[.625rem]` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

