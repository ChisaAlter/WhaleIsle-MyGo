# chat-typing-indicator

Source: `apps/ui/registry/default/chat/chat-typing-indicator.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatTypingIndicator`, `ChatTypingIndicatorContent`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatTypingIndicator`

No own classes (re-export / wrapper of the primitive); prop defaults: `visible=true`.

### `ChatTypingIndicatorContent` → (no data-slot) · `<motion.div>` (line 50)

Attributes: `initial={{ height: 0, opacity: 0 }}`, `animate={isPresent ? { height: "auto", opacity: 1 } : { height: 0, opacity: 0 }}`, `transition={{ duration: reduceMotion ? 0 : 0.2, ease: [0.22, 1, 0.36, 1], }}`

#### base

```txt
overflow-hidden
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| overflow | `hidden` | `hidden` | `overflow-hidden` |

### `ChatTypingIndicatorContent` → `[data-slot=chat-typing-indicator]` · `<div>` (line 64)

Attributes: `role=status`

#### base

```txt
flex items-center gap-2.5
```

**element — default**

Key metrics: display `flex` · gap `10px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `10px` | `calc(var(--spacing) * 2.5)` | `gap-2.5` |

### `ChatTypingIndicatorContent` → (no data-slot) · `<motion.div>` (line 70)

Attributes: `aria-hidden=true`, `initial={{ transform: reduceMotion ? "scale(1)" : "scale(0.92)" }}`, `animate={{ transform: isPresent \|\| reduceMotion ? "scale(1)" : "scale(0.96)", }}`, `transition={{ duration: reduceMotion ? 0 : 0.2, ease: [0.22, 1, 0.36, 1], }}`

#### base

```txt
flex origin-bottom-left items-center gap-1 rounded-2xl rounded-es-md bg-muted/72 px-3 py-3.5 rtl:origin-bottom-right
```

**element — default**

Key metrics: display `flex` · padding-inline `12px` · padding-block `14px` · gap `4px` · border-radius `16px` (radius-2xl) · background-color `#00000007` / `#ffffff07` (muted/72%)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| transform-origin | `0 100%` | `0 100%` | `origin-bottom-left` |
| align-items | `center` | `center` | `items-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| border-end-start-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-es-md` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `bg-muted/72` |
| padding-inline | `12px` | `calc(var(--spacing) * 3)` | `px-3` |
| padding-block | `14px` | `calc(var(--spacing) * 3.5)` | `py-3.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| rtl | transform-origin | `100% 100%` | `rtl:origin-bottom-right` |

### `ChatTypingIndicatorContent` → (no data-slot) · `<motion.span>` (line 83)

Attributes: `initial={{ transform: "translateY(0px) scale(0.8)", opacity: 0.4, }}`, `transition={reduceMotion ? { duration: 0 } : { duration: 1.4, times: [0, 0.22, 0.44, 1], delay: dot * 0.16, repeat: Number.POSITIVE`

#### base

```txt
block size-1.5 rounded-full bg-muted-foreground
```

**element — default**

Key metrics: display `block` · height `6px` · width `6px` · border-radius `9999px` · background-color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `block` | `block` | `block` |
| width | `6px` | `calc(var(--spacing) * 1.5)` | `size-1.5` |
| height | `6px` | `calc(var(--spacing) * 1.5)` | `size-1.5` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#686868` / `#818181` | `muted-foreground` | `bg-muted-foreground` |

### `ChatTypingIndicatorContent` → (no data-slot) · `<span>` (line 117)

#### base

```txt
min-w-0 text-muted-foreground text-xs
```

**element — default**

Key metrics: min-width `0px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

