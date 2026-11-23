# chat-message-text

Source: `apps/ui/registry/default/chat/chat-message-text.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatMessageText`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMessageText` → (no data-slot) · `<p>` (line 18)

Prop defaults: `mentions=[]`

#### base

```txt
m-0 whitespace-pre-wrap break-words text-sm leading-[1.55]
```

**element — default**

Key metrics: font-size `14px` (text-sm) · line-height `21.7px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `0px` | `calc(var(--spacing) * 0)` | `m-0` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `21.7px` | `1.55` | `leading-[1.55]` |
| overflow-wrap | `break-word` | `break-word` | `break-words` |
| white-space | `pre-wrap` | `pre-wrap` | `whitespace-pre-wrap` |

### `ChatMessageText` → (no data-slot) · `<span>` (line 31)

Prop defaults: `mentions=[]`

#### base

```txt
font-medium
```

**element — default**

Key metrics: font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `500` | `font-weight-medium` | `font-medium` |

