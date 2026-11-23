# chat-collapsible-message

Source: `apps/ui/registry/default/chat/chat-collapsible-message.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`, `@/ui/collapsible`

Exports: `renderPlainText`, `ChatCollapsibleMessage`, `ExpandableChatMessage`

## Behavior & keyboard

- Collapsible long message with show more/less trigger (Collapsible semantics).

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `renderPlainText` → (no data-slot) · `<p>` (line 29)

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

### `ChatCollapsibleMessage` → (no data-slot) · `<div>` (line 52)

Prop defaults: `defaultOpen=false`, `renderText=renderPlainText`

#### base

```txt
min-w-0
```

**element — default**

Key metrics: min-width `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |

### `ExpandableChatMessage` → (no data-slot) · `<Collapsible>` (line 103)

Attributes: `open={open}`

#### base

```txt
flex min-w-0 flex-col
```

**element — default**

Key metrics: display `flex` · min-width `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |

### `ExpandableChatMessage` → (no data-slot) · `<div>` (line 108)

#### base

```txt
grid
```

**element — default**

Key metrics: display `grid`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |

### `ExpandableChatMessage` → `[data-slot=chat-collapsible-preview]` · `<div>` (line 109)

Attributes: `aria-hidden={open \|\| undefined}`

#### base

```txt
mask-[linear-gradient(to_bottom,black_calc(100%-3rem),transparent)] col-start-1 row-start-1 self-start
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| grid-row-start | `1` | `1` | `row-start-1` |
| align-self | `flex-start` | `flex-start` | `self-start` |
| mask-image | `linear-gradient(to bottom,black calc(100% - 48px),transparent)` | `linear-gradient(to bottom,black calc(100% - 3rem),transparent)` | `mask-[linear-gradient(to_bottom,black_calc(100%-3rem),transparent)]` |

#### when `open`

```txt
invisible
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| visibility | `hidden` | `hidden` | `invisible` |

### `ExpandableChatMessage` → (no data-slot) · `<CollapsiblePanel>` (line 119)

#### base

```txt
col-start-1 row-start-1 min-w-0 motion-reduce:transition-none
```

**element — default**

Key metrics: min-width `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| grid-row-start | `1` | `1` | `row-start-1` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |

### `ExpandableChatMessage` → (no data-slot) · `<div>` (line 120)

Attributes: `aria-hidden={!open \|\| undefined}`

#### when `!(open)`

```txt
invisible
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| visibility | `hidden` | `hidden` | `invisible` |

### `ExpandableChatMessage` → (no data-slot) · `<CollapsibleTrigger>` (line 128)

Attributes: `type=button`

#### base

```txt
mt-2 flex w-fit items-center gap-1 text-start text-xs outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring
```

**element — default**

Key metrics: display `flex` · width `fit-content` · gap `4px` · font-size `12px` (text-xs) · line-height `16px` · transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-top | `8px` | `calc(var(--spacing) * 2)` | `mt-2` |
| display | `flex` | `flex` | `flex` |
| width | `fit-content` | `fit-content` | `w-fit` |
| align-items | `center` | `center` | `items-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
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
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` / `0 0 0 2px #737373` | `focus-visible:ring-2` |

#### when `side === "outgoing"`

```txt
text-foreground/80 hover:text-foreground
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#262626cc` / `#f5f5f5cc` | `foreground/80%` | `text-foreground/80` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` / `0 0 0 2px #737373` | `focus-visible:ring-2` |
| hover | color | `#262626` / `#f5f5f5` | `hover:text-foreground` |

#### when `!(side === "outgoing")`

```txt
text-muted-foreground hover:text-foreground
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` / `0 0 0 2px #737373` | `focus-visible:ring-2` |
| hover | color | `#262626` / `#f5f5f5` | `hover:text-foreground` |

### `ExpandableChatMessage` → (no data-slot) · `<Icon>` (line 138)

Attributes: `aria-hidden=true`

#### base

```txt
size-3.5 shrink-0
```

**element — default**

Key metrics: height `14px` · width `14px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `14px` | `calc(var(--spacing) * 3.5)` | `size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `size-3.5` |
| flex-shrink | `0` | `0` | `shrink-0` |

