# chat-message-bubble

Source: `apps/ui/registry/default/chat/chat-message-bubble.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatMessageBubble`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMessageBubble` → `[data-slot=chat-message-bubble]` · `<ChatBubble>` (line 35)

Attributes: `align={side === "outgoing" ? "end" : "start"}`, `variant={side === "outgoing" ? "default" : "muted"}`, `grouped={!firstInGroup}`

Prop defaults: `firstInGroup=true`, `animateEntrance=false`, `highlighted=false`

#### when `highlighted`

```txt
origin-center
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `center` | `center` | `origin-center` |

#### when `!(highlighted) && side === "outgoing"`

```txt
origin-top-right rtl:origin-top-left
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `100% 0` | `100% 0` | `origin-top-right` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| rtl | transform-origin | `0 0` | `rtl:origin-top-left` |

#### when `!(highlighted) && !(side === "outgoing")`

```txt
origin-top-left rtl:origin-top-right
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform-origin | `0 0` | `0 0` | `origin-top-left` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| rtl | transform-origin | `100% 0` | `rtl:origin-top-right` |

### `ChatMessageBubble` → (no data-slot) · `<motion.div>` (line 75)

Attributes: `aria-hidden=true`, `initial={{ opacity: 0 }}`, `animate={{ opacity: [0, 1, 0] }}`, `transition={{ duration: 0.42, times: [0, 0.28, 1], ease: "easeOut" }}`

Prop defaults: `firstInGroup=true`, `animateEntrance=false`, `highlighted=false`

#### base

```txt
pointer-events-none absolute inset-0 rounded-[inherit] ring-2 ring-foreground/25 ring-offset-2 ring-offset-background
```

**element — default**

Key metrics: border-radius `inherit` · box-shadow `0 0 0 2px #ffffff, 0 0 0 4px #26262640` / `0 0 0 2px #141414, 0 0 0 4px #f5f5f540`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| box-shadow | `0 0 0 2px #ffffff, 0 0 0 4px #26262640` / `0 0 0 2px #141414, 0 0 0 4px #f5f5f540` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `ring-2` |

