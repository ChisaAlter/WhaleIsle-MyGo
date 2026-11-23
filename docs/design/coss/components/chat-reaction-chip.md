# chat-reaction-chip

Source: `apps/ui/registry/default/chat/chat-reaction-chip.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `@/ui/popover`, `@/ui/toggle`

Exports: `ChatReactionChip`

## Behavior & keyboard

- Toggle-like chip (`aria-pressed`) showing reaction count; `Enter`/`Space` toggles. _(inferred)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatReactionChip` → `[data-slot=chat-reaction-chip]` · `<Toggle>` (line 70)

Attributes: `variant=outline`, `size=sm`, `aria-label={label}`, `pressed={selected}`

Prop defaults: `animateEntrance=false`

#### base

```txt
h-6 min-w-0 gap-1 rounded-full px-2 text-xs shadow-none transition-transform duration-150 [-webkit-touch-callout:none] before:rounded-full active:scale-95 data-pressed:border-primary/48 data-pressed:bg-primary/12 motion-reduce:scale-100 motion-reduce:transition-none sm:h-6 sm:min-w-0 sm:text-xs dark:data-pressed:bg-primary/8
```

**element — default**

Key metrics: height `24px` · min-width `0px` · padding-inline `8px` · gap `4px` · border-radius `9999px` · font-size `12px` (text-xs) · line-height `16px` · box-shadow `none` · transition-property `transform, translate, scale, rotate` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `24px` | `calc(var(--spacing) * 6)` | `sm:h-6` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `sm:min-w-0` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| font-size | `12px` | `text-xs` | `sm:text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `sm:text-xs` |
| box-shadow | `none` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-none` |
| transition-property | `transform, translate, scale, rotate` | `transform, translate, scale, rotate` | `transition-transform` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-transform` |
| transition-duration | `150ms` | `150ms` | `duration-150` |
| -webkit-touch-callout | `none` | `none` | `[-webkit-touch-callout:none]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| pressed | border-color | `#2626267a` / `#f5f5f57a` | `data-pressed:border-primary/48` |
| pressed | scale | `95% 95%` | `active:scale-95` |
| pressed | background-color | `#2626261f` / `#f5f5f514` | `data-pressed:bg-primary/12` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

### `ChatReactionChip` → (no data-slot) · `<motion.span>` (line 89)

Attributes: `initial={false}`, `transition={{ duration: reduceMotion ? 0 : 0.44, times: [0, 0.35, 0.65, 1], ease: "easeOut", }}`

Prop defaults: `animateEntrance=false`

#### base

```txt
inline-flex
```

**element — default**

Key metrics: display `inline-flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |

### `ChatReactionChip` → (no data-slot) · `<span>` (line 112)

Prop defaults: `animateEntrance=false`

#### base

```txt
tabular-nums
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-variant-numeric | `tabular-nums` | `var(--tw-ordinal,) var(--tw-slashed-zero,) var(--tw-numeric-figure,) var(--tw-numeric-spacing,) var(--tw-numer` | `tabular-nums` |

### `ChatReactionChip` → (no data-slot) · `<span>` (line 136)

Prop defaults: `animateEntrance=false`

#### base

```txt
inline-flex
```

**element — default**

Key metrics: display `inline-flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |

### `ChatReactionChip` → (no data-slot) · `<PopoverPopup>` (line 198)

Attributes: `anchor={anchorRef}`, `align={popupAlign}`, `initialFocus={false}`, `finalFocus={false}`, `aria-label={reactorLabel}`

Prop defaults: `animateEntrance=false`

#### base

```txt
w-52 **:data-[slot=popover-viewport]:p-1.5
```

**element — default**

Key metrics: width `208px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `208px` | `calc(var(--spacing) * 52)` | `w-52` |

**`*[data-slot="popover-viewport"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding | `6px` | `calc(var(--spacing) * 1.5)` | `**:data-[slot=popover-viewport]:p-1.5` |

### `ChatReactionChip` → (no data-slot) · `<motion.span>` (line 213)

Attributes: `initial={animateEntrance && !reduceMotion ? { opacity: 0, transform: "scale(0.55)" } : false}`, `animate={{ opacity: 1, transform: "scale(1)" }}`, `transition={reduceMotion ? { duration: 0 } : { transform: { type: "spring", duration: 0.44, bounce: 0.45 }, opacity: { duration: 0.`

Prop defaults: `animateEntrance=false`

#### base

```txt
inline-flex origin-top
```

**element — default**

Key metrics: display `inline-flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| transform-origin | `top` | `top` | `origin-top` |

#### when `animateEntrance && !reduceMotion`

```txt
will-change-transform
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| will-change | `transform` | `transform` | `will-change-transform` |

