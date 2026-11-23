# chat-delivery-indicator

Source: `apps/ui/registry/default/chat/chat-delivery-indicator.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`

Exports: `ChatDeliveryIndicator`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatDeliveryIndicator` → `[data-slot=chat-delivery-indicator]` · `<span>` (line 30)

Attributes: `role=img`, `aria-label={label}`, `title={label}`

### `ChatDeliveryIndicator` → (no data-slot) · `<Icon>` (line 36)

Attributes: `aria-hidden=true`

#### base

```txt
size-3
```

**element — default**

Key metrics: height `12px` · width `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `12px` | `calc(var(--spacing) * 3)` | `size-3` |
| height | `12px` | `calc(var(--spacing) * 3)` | `size-3` |

#### when `status === "read"`

```txt
text-sky-600 dark:text-sky-400
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#0084d1` / `#00bcff` | `color-sky-600` | `text-sky-600` |

#### when `status === "failed"`

```txt
text-destructive
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#fb2c36` / `#fb414a` | `destructive` | `text-destructive` |

