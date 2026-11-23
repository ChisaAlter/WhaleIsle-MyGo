# chat-reply-reference

Source: `apps/ui/registry/default/chat/chat-reply-reference.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`

Exports: `ChatReplyReference`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatReplyReference` → `[data-slot=chat-reply-reference]` · `<button>` (line 22)

Attributes: `type=button`

#### base

```txt
group/reply-reference flex min-w-0 items-start gap-1.5 text-start text-xs outline-none transition-colors focus-visible:rounded-sm focus-visible:text-primary focus-visible:ring-2 focus-visible:ring-current enabled:hover:text-primary motion-reduce:transition-none
```

Classes that produce no CSS (custom markers / group names): `group/reply-reference`

**element — default**

Key metrics: display `flex` · min-width `0px` · gap `6px` · font-size `12px` (text-xs) · line-height `16px` · transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |
| text-align | `start` | `start` | `text-start` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| transition-property | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from,` | `transition-colors` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-colors` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-colors` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | border-radius | `6px` | `focus-visible:rounded-sm` |
| focus-visible | color | `#262626` / `#f5f5f5` | `focus-visible:text-primary` |
| focus-visible | box-shadow | `0 0 0 2px currentcolor` | `focus-visible:ring-2` |
| hover | color | `#262626` / `#f5f5f5` | `enabled:hover:text-primary` |

### `ChatReplyReference` → (no data-slot) · `<CornerDownRightIcon>` (line 31)

Attributes: `aria-hidden=true`

#### base

```txt
mt-0.5 size-3.5 shrink-0 opacity-64 transition-opacity group-hover/reply-reference:opacity-100 group-focus-visible/reply-reference:opacity-100
```

**element — default**

Key metrics: height `14px` · width `14px` · opacity `64%` · transition-property `opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-top | `2px` | `calc(var(--spacing) * 0.5)` | `mt-0.5` |
| width | `14px` | `calc(var(--spacing) * 3.5)` | `size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `size-3.5` |
| flex-shrink | `0` | `0` | `shrink-0` |
| opacity | `64%` | `64%` | `opacity-64` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-opacity` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group\/reply-reference:hover) | opacity | `100%` | `group-hover/reply-reference:opacity-100` |
| in(.group\/reply-reference:focus-visible) | opacity | `100%` | `group-focus-visible/reply-reference:opacity-100` |

### `ChatReplyReference` → (no data-slot) · `<span>` (line 35)

#### base

```txt
flex min-w-0 flex-col gap-0.5
```

**element — default**

Key metrics: display `flex` · min-width `0px` · gap `2px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-0.5` |

### `ChatReplyReference` → (no data-slot) · `<span>` (line 36)

#### base

```txt
font-medium
```

**element — default**

Key metrics: font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `500` | `font-weight-medium` | `font-medium` |

### `ChatReplyReference` → (no data-slot) · `<span>` (line 37)

#### base

```txt
line-clamp-2 text-foreground/80
```

**element — default**

Key metrics: display `-webkit-box` · color `#262626cc` / `#f5f5f5cc` (foreground/80%)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| overflow | `hidden` | `hidden` | `line-clamp-2` |
| display | `-webkit-box` | `-webkit-box` | `line-clamp-2` |
| -webkit-box-orient | `vertical` | `vertical` | `line-clamp-2` |
| -webkit-line-clamp | `2` | `2` | `line-clamp-2` |
| color | `#262626cc` / `#f5f5f5cc` | `foreground/80%` | `text-foreground/80` |

