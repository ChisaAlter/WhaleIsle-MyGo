# chat-unread-divider

Source: `apps/ui/registry/default/chat/chat-unread-divider.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatUnreadDivider`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatUnreadDivider` → `[data-slot=chat-unread-divider]` · `<div>` (line 12)

#### base

```txt
my-4 flex items-center gap-3 font-medium text-foreground text-xs
```

**element — default**

Key metrics: display `flex` · gap `12px` · color `#262626` / `#f5f5f5` (foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-block | `16px` | `calc(var(--spacing) * 4)` | `my-4` |
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `12px` | `calc(var(--spacing) * 3)` | `gap-3` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

### `ChatUnreadDivider` → (no data-slot) · `<span>` (line 20)

Attributes: `aria-hidden=true`

#### base

```txt
h-px flex-1 bg-input
```

**element — default**

Key metrics: height `1px` · background-color `#0000001a` / `#ffffff14` (input)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `1px` | `1px` | `h-px` |
| flex | `1` | `1` | `flex-1` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

### `ChatUnreadDivider` → (no data-slot) · `<span>` (line 22)

Attributes: `aria-hidden=true`

#### base

```txt
h-px flex-1 bg-input
```

**element — default**

Key metrics: height `1px` · background-color `#0000001a` / `#ffffff14` (input)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `1px` | `1px` | `h-px` |
| flex | `1` | `1` | `flex-1` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

