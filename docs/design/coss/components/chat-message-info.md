# chat-message-info

Source: `apps/ui/registry/default/chat/chat-message-info.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatMessageInfo`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMessageInfo` → `[data-slot=chat-message-info]` · `<div>` (line 24)

### `ChatMessageInfo` → (no data-slot) · `<div>` (line 25)

#### base

```txt
mb-7 flex flex-col items-end gap-1
```

**element — default**

Key metrics: display `flex` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-bottom | `28px` | `calc(var(--spacing) * 7)` | `mb-7` |
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `flex-end` | `flex-end` | `items-end` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

### `ChatMessageInfo` → (no data-slot) · `<div>` (line 26)

#### base

```txt
w-fit min-w-0 max-w-[83%]
```

**element — default**

Key metrics: width `fit-content` · min-width `0px` · max-width `83%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `fit-content` | `fit-content` | `w-fit` |
| max-width | `83%` | `83%` | `max-w-[83%]` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |

### `ChatMessageInfo` → (no data-slot) · `<div>` (line 29)

#### base

```txt
flex flex-col gap-7
```

**element — default**

Key metrics: display `flex` · gap `28px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `28px` | `calc(var(--spacing) * 7)` | `gap-7` |

