# chat-message-actions

Source: `apps/ui/registry/default/chat/chat-message-actions.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`, `@/ui/button`, `@/ui/menu`, `@/ui/toolbar`, `@/ui/tooltip`

Exports: `ChatMessageActionToolbar`

## Behavior & keyboard

- Hover/focus-revealed action toolbar per message; reachable by keyboard focus within the message (`group/message:focus-within`). _(inferred from classes)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMessageActionToolbar` → `[data-slot=chat-message-action-toolbar]` · `<Toolbar>` (line 55)

Attributes: `aria-label={toolbarLabel}`

#### base

```txt
z-10 min-w-max flex-none items-center gap-0 self-center rounded-full bg-transparent p-0.5 opacity-0 pointer-coarse:opacity-100 transition-opacity group-hover/message:opacity-100 has-data-popup-open:opacity-100 has-focus-visible:opacity-100 motion-reduce:transition-none
```

**element — default**

Key metrics: min-width `max-content` · padding `2px` · gap `0px` · border-radius `9999px` · background-color `#00000000` · opacity `0%` · z-index `10` · transition-property `opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `10` | `10` | `z-10` |
| min-width | `max-content` | `max-content` | `min-w-max` |
| flex | `none` | `none` | `flex-none` |
| align-items | `center` | `center` | `items-center` |
| gap | `0px` | `calc(var(--spacing) * 0)` | `gap-0` |
| align-self | `center` | `center` | `self-center` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |
| padding | `2px` | `calc(var(--spacing) * 0.5)` | `p-0.5` |
| opacity | `0%` | `0%` | `opacity-0` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-opacity` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group\/message:hover) | opacity | `100%` | `group-hover/message:opacity-100` |
| has([data-popup-open]) | opacity | `100%` | `has-data-popup-open:opacity-100` |
| focus-visible | opacity | `100%` | `has-focus-visible:opacity-100` |

**element — mobile differences**: opacity `100%`

### `ChatMessageActionToolbar` → (no data-slot) · `<Button>` (line 68)

Attributes: `size=icon-xs`, `variant=ghost`, `aria-label={replyLabel}`

#### base

```txt
pointer-coarse:size-8 rounded-full before:rounded-full
```

**element — default**

Key metrics: border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**element — mobile differences**: width `32px` · height `32px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

### `ChatMessageActionToolbar` → (no data-slot) · `<Button>` (line 106)

Attributes: `size=icon-xs`, `variant=ghost`, `aria-label={moreLabel}`, `data-message-actions=`

#### base

```txt
pointer-coarse:size-8 rounded-full before:rounded-full
```

**element — default**

Key metrics: border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**element — mobile differences**: width `32px` · height `32px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

