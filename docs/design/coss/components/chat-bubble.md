# chat-bubble

Source: `apps/ui/registry/default/chat/chat-bubble.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `ChatBubble`, `ChatBubbleReactions`

## Behavior & keyboard

- Presentational message bubble.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatBubble` → `[data-slot=chat-bubble]` · `useRender <div>` (line 22)

Attributes: `data-align={align}`, `data-variant={variant}`, `data-grouped={grouped \|\| undefined}`

Prop defaults: `align="start"`, `variant="default"`, `grouped=false`

#### base

```txt
relative flex min-w-0 flex-col gap-2 rounded-2xl px-3.5 py-2.5 text-foreground
```

**element — default**

Key metrics: display `flex` · min-width `0px` · padding-inline `14px` · padding-block `10px` · gap `8px` · border-radius `16px` (radius-2xl) · color `#262626` / `#f5f5f5` (foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| padding-inline | `14px` | `calc(var(--spacing) * 3.5)` | `px-3.5` |
| padding-block | `10px` | `calc(var(--spacing) * 2.5)` | `py-2.5` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

#### when `variant === "default"`

```txt
bg-foreground/12
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#2626261f` / `#f5f5f51f` | `foreground/12%` | `bg-foreground/12` |

#### when `!(variant === "default")`

```txt
bg-foreground/4
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#2626260a` / `#f5f5f50a` | `foreground/4%` | `bg-foreground/4` |

#### when `!grouped && align === "end"`

```txt
rounded-se-md
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-se-md` |

#### when `!grouped && !(align === "end")`

```txt
rounded-ss-md
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-ss-md` |

### `ChatBubbleReactions` → `[data-slot=chat-bubble-reactions]` · `<div>` (line 47)

#### base

```txt
flex flex-wrap gap-1 px-0.5
```

**element — default**

Key metrics: display `flex` · padding-inline `2px` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-wrap | `wrap` | `wrap` | `flex-wrap` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| padding-inline | `2px` | `calc(var(--spacing) * 0.5)` | `px-0.5` |

