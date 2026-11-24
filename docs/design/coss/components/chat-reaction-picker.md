# chat-reaction-picker

Source: `apps/ui/registry/default/chat/chat-reaction-picker.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: `lucide-react`, `@/ui/button`, `@/ui/popover`, `@/ui/toolbar`, `@/ui/tooltip`

Exports: `ChatReactionPicker`

## Behavior & keyboard

- Popover/menu of emoji reactions; arrow-key navigation follows the underlying menu/popover primitive. _(inferred)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatReactionPicker` → (no data-slot) · `<Button>` (line 52)

Attributes: `size=icon-xs`, `variant=ghost`, `aria-label={triggerLabel}`

Prop defaults: `inToolbar=false`

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

### `ChatReactionPicker` → (no data-slot) · `<HeartIcon>` (line 71)

Attributes: `aria-hidden=true`

Prop defaults: `inToolbar=false`

#### base

```txt
size-3.5
```

**element — default**

Key metrics: height `14px` · width `14px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `14px` | `calc(var(--spacing) * 3.5)` | `size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `size-3.5` |

### `ChatReactionPicker` → (no data-slot) · `<PopoverPopup>` (line 75)

Attributes: `anchor={popupAnchor}`, `side=top`, `align={popupAlign}`, `aria-label={popupLabel}`, `initialFocus={(interaction) => interaction === "keyboard"}`, `finalFocus={(interaction) => interaction === "keyboard"}`

Prop defaults: `inToolbar=false`

#### base

```txt
transform-gpu rounded-full before:rounded-full **:data-[slot=popover-viewport]:p-1
```

**element — default**

Key metrics: border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transform | `translateZ(0)` | `translateZ(0) var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,)` | `transform-gpu` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

**`*[data-slot="popover-viewport"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding | `4px` | `calc(var(--spacing) * 1)` | `**:data-[slot=popover-viewport]:p-1` |

### `ChatReactionPicker` → (no data-slot) · `<div>` (line 84)

Prop defaults: `inToolbar=false`

#### base

```txt
flex
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |

### `ChatReactionPicker` → (no data-slot) · `<Button>` (line 86)

Attributes: `aria-label={label}`, `aria-pressed={selectedEmojis.includes(emoji)}`, `variant=ghost`, `size=icon-lg`

Prop defaults: `inToolbar=false`

#### base

```txt
rounded-full before:rounded-full
```

**element — default**

Key metrics: border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-full` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |

### `ChatReactionPicker` → (no data-slot) · `<span>` (line 101)

Attributes: `aria-hidden=true`

Prop defaults: `inToolbar=false`

#### base

```txt
text-lg leading-none
```

**element — default**

Key metrics: font-size `18px` (text-lg) · line-height `18px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `18px` | `text-lg` | `text-lg` |
| line-height | `18px` | `1` | `leading-none` |

