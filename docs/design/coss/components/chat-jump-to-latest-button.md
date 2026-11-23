# chat-jump-to-latest-button

Source: `apps/ui/registry/default/chat/chat-jump-to-latest-button.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`, `@/ui/button`

Exports: `ChatJumpToLatestButton`

## Behavior & keyboard

- Button shown when scrolled away from the bottom; activates scroll to latest.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatJumpToLatestButton` → `[data-slot=chat-jump-to-latest]` · `<div>` (line 21)

#### base

```txt
rounded-full bg-popover/90 backdrop-blur-xs
```

**element — default**

Key metrics: border-radius `9999px` · background-color `#ffffffe6` / `#1d1d1de6` (popover/90%)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#ffffffe6` / `#1d1d1de6` | `popover/90%` | `bg-popover/90` |
| -webkit-backdrop-filter | `blur(4px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-xs` |
| backdrop-filter | `blur(4px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-xs` |

### `ChatJumpToLatestButton` → (no data-slot) · `<Button>` (line 25)

Attributes: `variant=outline`, `size=xs`

#### base

```txt
pointer-events-auto gap-2 rounded-full before:rounded-full
```

**element — default**

Key metrics: gap `8px` · border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `auto` | `auto` | `pointer-events-auto` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

