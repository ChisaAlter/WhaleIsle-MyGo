# chat-message

Source: `apps/ui/registry/default/chat/chat-message.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatMessage`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMessage` → (no data-slot) · `<ChatMessageHeader>` (line 67)

Prop defaults: `grouped=false`, `lifted=false`, `bubble=true`

#### when `lifted`

```txt
invisible
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| visibility | `hidden` | `hidden` | `invisible` |

### `ChatMessage` → (no data-slot) · `<ChatMessageAvatar>` (line 72)

Prop defaults: `grouped=false`, `lifted=false`, `bubble=true`

#### when `lifted`

```txt
invisible
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| visibility | `hidden` | `hidden` | `invisible` |

### `ChatMessage` → (no data-slot) · `<div>` (line 78)

Prop defaults: `grouped=false`, `lifted=false`, `bubble=true`

#### base

```txt
flex w-full min-w-0 items-center gap-2
```

**element — default**

Key metrics: display `flex` · width `100%` · min-width `0px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

#### when `own`

```txt
flex-row-reverse
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-direction | `row-reverse` | `row-reverse` | `flex-row-reverse` |

### `ChatMessage` → (no data-slot) · `<div>` (line 84)

Prop defaults: `grouped=false`, `lifted=false`, `bubble=true`

#### base

```txt
relative min-w-0 max-w-[83%] shrink
```

**element — default**

Key metrics: min-width `0px` · max-width `83%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| max-width | `83%` | `83%` | `max-w-[83%]` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-shrink | `1` | `1` | `shrink` |

### `ChatMessage` → (no data-slot) · `<ChatMessageBubble>` (line 85)

Attributes: `side={side}`, `firstInGroup={!grouped}`, `animateEntrance={animateEntrance}`, `highlighted={highlighted}`

Prop defaults: `grouped=false`, `lifted=false`, `bubble=true`

Dynamic className inputs (not statically resolvable): `bubbleProps?.className`

#### when `lifted`

```txt
invisible
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| visibility | `hidden` | `hidden` | `invisible` |

