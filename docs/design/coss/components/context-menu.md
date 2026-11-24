# Context Menu

Source: `apps/ui/registry/default/ui/context-menu.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/context-menu.mdx` · Base UI: https://base-ui.com/react/components/context-menu#api-reference

> A menu that appears at the pointer on right click or long press.

Depends on: `@base-ui/react/context-menu`, `lucide-react`

Exports: `ContextMenuTrigger`, `ContextMenuPopup`, `ContextMenuGroup`, `ContextMenuItem`, `ContextMenuLinkItem`, `ContextMenuCheckboxItem`, `ContextMenuRadioGroup`, `ContextMenuRadioItem`, `ContextMenuGroupLabel`, `ContextMenuSeparator`, `ContextMenuShortcut`, `ContextMenuSub`, `ContextMenuSubTrigger`, `ContextMenuSubPopup`

## Behavior & keyboard

- Opens on right-click / long-press (touch) at pointer position; same keyboard model as Menu: `ArrowUp/Down` move highlight, `ArrowRight` opens submenu, `ArrowLeft`/`Escape` close, `Enter`/`Space` activate, typeahead by label.
- Popup is portalled, positioned by Base UI Positioner (`side`/`align`/`sideOffset` props, CSS vars `--anchor-width`, `--available-height`, `--transform-origin`). Enter/exit animate via `data-starting-style` / `data-ending-style` attributes set by Base UI for one frame / until transitionend. _(inferred: Base UI primitive behavior; not restated in coss docs)_

Docs mentions:

- ### ContextMenuShortcut
- Displays keyboard shortcut text. Custom component (not a Base UI wrapper).

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ContextMenuTrigger` → `[data-slot=context-menu-trigger]` · `<ContextMenuPrimitive.Trigger>` (line 20)

### `ContextMenuPopup` → `[data-slot=context-menu-positioner]` · `<ContextMenuPrimitive.Positioner>` (line 50)

Attributes: `align={align}`, `alignOffset={alignOffset}`, `anchor={anchor}`, `side={side}`, `sideOffset={sideOffset}`

Prop defaults: `sideOffset=4`, `align="center"`, `side="bottom"`

#### base

```txt
z-50
```

**element — default**

Key metrics: z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `50` | `50` | `z-50` |

### `ContextMenuPopup` → `[data-slot=context-menu-popup]` · `<ContextMenuPrimitive.Popup>` (line 59)

Prop defaults: `sideOffset=4`, `align="center"`, `side="bottom"`

#### base

```txt
relative flex not-[class*='w-']:min-w-32 origin-(--transform-origin) rounded-lg border bg-popover not-dark:bg-clip-padding shadow-lg/5 outline-none before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] focus:outline-none dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · min-width `128px` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `#ffffff` / `#1d1d1d` (popover) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| transform-origin | `var(--transform-origin)` | `runtime var --transform-origin` | `origin-(--transform-origin)` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| outline-style | `none` | `none` | `outline-none` |
| min-width | `128px` | `calc(var(--spacing) * 32)` | `not-[class*='w-']:min-w-32` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| userclass(w-) | min-width | `(unset)` | `(removed)` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `ContextMenuPopup` → (no data-slot) · `<div>` (line 67)

Prop defaults: `sideOffset=4`, `align="center"`, `side="bottom"`

#### base

```txt
max-h-(--available-height) w-full overflow-y-auto p-1
```

**element — default**

Key metrics: max-height `var(--available-height)` (runtime var --available-height) · width `100%` · padding `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| max-height | `var(--available-height)` | `runtime var --available-height` | `max-h-(--available-height)` |
| width | `100%` | `100%` | `w-full` |
| overflow-y | `auto` | `auto` | `overflow-y-auto` |
| padding | `4px` | `calc(var(--spacing) * 1)` | `p-1` |

### `ContextMenuGroup` → `[data-slot=context-menu-group]` · `<ContextMenuPrimitive.Group>` (line 80)

### `ContextMenuItem` → `[data-slot=context-menu-item]` · `<ContextMenuPrimitive.Item>` (line 94)

Attributes: `data-inset={inset}`, `data-variant={variant}`

Prop defaults: `variant="default"`

#### base

```txt
flex min-h-8 cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-inset:ps-8 data-[variant=destructive]:text-destructive-foreground data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&>svg:not([class*='opacity-'])]:opacity-80 [&>svg:not([class*='size-'])]:size-4.5 sm:[&>svg:not([class*='size-'])]:size-4 [&>svg]:pointer-events-none [&>svg]:-mx-0.5 [&>svg]:shrink-0
```

**element — default**

Key metrics: display `flex` · min-height `28px` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |
| [data-inset] | padding-inline-start | `32px` | `data-inset:ps-8` |
| [data-variant=destructive] | color | `#c10007` / `#ff6467` | `data-[variant=destructive]:text-destructive-foreground` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&>svg]:pointer-events-none` |
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&>svg]:-mx-0.5` |
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

**`>svg:not([class*='opacity-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `[&>svg:not([class*='opacity-'])]:opacity-80` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&>svg:not([class*='size-'])]:size-4` |

**`>svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ContextMenuLinkItem` → `[data-slot=context-menu-link-item]` · `<ContextMenuPrimitive.LinkItem>` (line 118)

Attributes: `closeOnClick={closeOnClick}`, `data-inset={inset}`, `data-variant={variant}`

Prop defaults: `variant="default"`, `closeOnClick=true`

#### base

```txt
flex min-h-8 cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-inset:ps-8 data-[variant=destructive]:text-destructive-foreground data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&>svg:not([class*='opacity-'])]:opacity-80 [&>svg:not([class*='size-'])]:size-4.5 sm:[&>svg:not([class*='size-'])]:size-4 [&>svg]:pointer-events-none [&>svg]:-mx-0.5 [&>svg]:shrink-0
```

**element — default**

Key metrics: display `flex` · min-height `28px` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |
| [data-inset] | padding-inline-start | `32px` | `data-inset:ps-8` |
| [data-variant=destructive] | color | `#c10007` / `#ff6467` | `data-[variant=destructive]:text-destructive-foreground` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&>svg]:pointer-events-none` |
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&>svg]:-mx-0.5` |
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

**`>svg:not([class*='opacity-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `[&>svg:not([class*='opacity-'])]:opacity-80` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&>svg:not([class*='size-'])]:size-4` |

**`>svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ContextMenuCheckboxItem` → `[data-slot=context-menu-checkbox-item]` · `<ContextMenuPrimitive.CheckboxItem>` (line 142)

Attributes: `checked={checked}`

Prop defaults: `variant="default"`

#### base

```txt
grid min-h-8 in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)] cursor-default items-center gap-2 rounded-sm py-1 ps-2 text-base text-foreground outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `grid` · min-height `28px` · padding-block `4px` · padding-inline-start `8px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| padding-inline-start | `8px` | `calc(var(--spacing) * 2)` | `ps-2` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-side=none]) | min-width | `calc(var(--anchor-width) + 20px)` | `in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)]` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### when `variant === "switch"`

```txt
grid-cols-[1fr_auto] gap-4 pe-1.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| gap | `16px` | `calc(var(--spacing) * 4)` | `gap-4` |
| grid-template-columns | `1fr auto` | `1fr auto` | `grid-cols-[1fr_auto]` |
| padding-inline-end | `6px` | `calc(var(--spacing) * 1.5)` | `pe-1.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-side=none]) | min-width | `calc(var(--anchor-width) + 20px)` | `in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)]` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

#### when `!(variant === "switch")`

```txt
grid-cols-[.75rem_1fr] pe-4
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-template-columns | `12px 1fr` | `.75rem 1fr` | `grid-cols-[.75rem_1fr]` |
| padding-inline-end | `16px` | `calc(var(--spacing) * 4)` | `pe-4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-side=none]) | min-width | `calc(var(--anchor-width) + 20px)` | `in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)]` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ContextMenuCheckboxItem` → (no data-slot) · `<span>` (line 156)

Prop defaults: `variant="default"`

#### base

```txt
col-start-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |

### `ContextMenuCheckboxItem` → (no data-slot) · `<ContextMenuPrimitive.CheckboxItemIndicator>` (line 157)

Attributes: `keepMounted=true`

Prop defaults: `variant="default"`

#### base

```txt
inset-shadow-[0_1px_--theme(--color-black/4%)] inline-flex h-[calc(var(--thumb-size)+2px)] w-[calc(var(--thumb-size)*2-2px)] shrink-0 items-center rounded-full p-px outline-none transition-[background-color,box-shadow] duration-200 [--thumb-size:--spacing(4)] focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background data-checked:bg-primary data-unchecked:bg-input data-disabled:opacity-64 sm:[--thumb-size:--spacing(3)]
```

**element — default**

Key metrics: display `inline-flex` · height `14px` · width `22px` · padding `1px` · border-radius `9999px` · box-shadow `inset 0 1px #0000000a` · transition-property `background-color, box-shadow` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| height | `14px` | `calc(var(--thumb-size) + 2px)` | `h-[calc(var(--thumb-size)+2px)]` |
| width | `22px` | `calc(var(--thumb-size) * 2 - 2px)` | `w-[calc(var(--thumb-size)*2-2px)]` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| padding | `1px` | `1px` | `p-px` |
| box-shadow | `inset 0 1px #0000000a` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `inset-shadow-[0_1px_--theme(--color-black/4%)]` |
| transition-property | `background-color, box-shadow` | `background-color,box-shadow` | `transition-[background-color,box-shadow]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[background-color,box-shadow]` |
| transition-duration | `200ms` | `200ms` | `duration-200` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `inset 0 1px #0000000a, 0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1` / `inset 0 1px #0000000a, 0 0 0 1px #141414, 0 0 0 3px #737373` | `focus-visible:ring-2` |
| checked | background-color | `#262626` / `#f5f5f5` | `data-checked:bg-primary` |
| unchecked | background-color | `#0000001a` / `#ffffff14` | `data-unchecked:bg-input` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |

**element — mobile differences**: height `18px` · width `30px`

### `ContextMenuCheckboxItem` → (no data-slot) · `<span>` (line 161)

Prop defaults: `variant="default"`

#### base

```txt
pointer-events-none block aspect-square h-full in-[[data-slot=context-menu-checkbox-item][data-checked]]:origin-[var(--thumb-size)_50%] origin-left in-[[data-slot=context-menu-checkbox-item][data-checked]]:translate-x-[calc(var(--thumb-size)-4px)] in-[[data-slot=context-menu-checkbox-item]:active]:not-data-disabled:scale-x-110 in-[[data-slot=context-menu-checkbox-item]:active]:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.10)] rounded-(--thumb-size) bg-background shadow-sm/5 will-change-transform [transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s] rtl:in-[[data-slot=context-menu-checkbox-item][data-checked]]:origin-left rtl:origin-right rtl:in-[[data-slot=context-menu-checkbox-item][data-checked]]:-translate-x-[calc(var(--thumb-size)-4px)]
```

**element — default**

Key metrics: display `block` · height `100%` · border-radius `12px` (runtime var --thumb-size) · background-color `#ffffff` / `#141414` (background) · box-shadow `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| display | `block` | `block` | `block` |
| aspect-ratio | `1 / 1` | `1 / 1` | `aspect-square` |
| height | `100%` | `100%` | `h-full` |
| transform-origin | `0` | `0` | `origin-left` |
| border-radius | `12px` | `runtime var --thumb-size` | `rounded-(--thumb-size)` |
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |
| box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-sm/5` |
| will-change | `transform` | `transform` | `will-change-transform` |
| transition | `translate 150ms, border-radius 150ms, scale 100ms 100ms, transform-origin 150ms` | `translate .15s,border-radius .15s,scale .1s .1s,transform-origin .15s` | `[transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=context-menu-checkbox-item][data-checked]))) | transform-origin | `12px 50%` | `in-[[data-slot=context-menu-checkbox-item][data-checked]]:origin-[var(--thumb-size)_50%]` |
| in(:where(*:is([data-slot=context-menu-checkbox-item][data-checked]))) | translate | `8px 0` | `in-[[data-slot=context-menu-checkbox-item][data-checked]]:translate-x-[calc(var(--thumb-size)-4px)]` |
| in(:where(*:is([data-slot=context-menu-checkbox-item]:active))) | border-radius | `12px/13.2px` | `in-[[data-slot=context-menu-checkbox-item]:active]:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.10)]` |
| in(:where(*:is([data-slot=context-menu-checkbox-item]:active))) | scale | `110% 1` | `in-[[data-slot=context-menu-checkbox-item]:active]:not-data-disabled:scale-x-110` |
| rtl | transform-origin | `100%` | `rtl:origin-right` |
| in(:where(*:is([data-slot=context-menu-checkbox-item][data-checked]))) + rtl | translate | `-8px 0` | `rtl:in-[[data-slot=context-menu-checkbox-item][data-checked]]:-translate-x-[calc(var(--thumb-size)-4px)]` |

**element — mobile differences**: border-radius `16px`

### `ContextMenuCheckboxItem` → (no data-slot) · `<ContextMenuPrimitive.CheckboxItemIndicator>` (line 166)

Prop defaults: `variant="default"`

#### base

```txt
col-start-1 -ms-0.5
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| margin-inline-start | `-2px` | `calc(var(--spacing) * -0.5)` | `-ms-0.5` |

### `ContextMenuCheckboxItem` → (no data-slot) · `<span>` (line 182)

Prop defaults: `variant="default"`

#### base

```txt
col-start-2
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |

### `ContextMenuRadioGroup` → `[data-slot=context-menu-radio-group]` · `<ContextMenuPrimitive.RadioGroup>` (line 193)

### `ContextMenuRadioItem` → `[data-slot=context-menu-radio-item]` · `<ContextMenuPrimitive.RadioItem>` (line 206)

#### base

```txt
grid min-h-8 in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)] cursor-default grid-cols-[.75rem_1fr] items-center gap-2 rounded-sm py-1 ps-2 pe-4 text-base text-foreground outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**element — default**

Key metrics: display `grid` · min-height `28px` · padding-block `4px` · padding-inline-start `8px` · padding-inline-end `16px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| cursor | `default` | `default` | `cursor-default` |
| grid-template-columns | `12px 1fr` | `.75rem 1fr` | `grid-cols-[.75rem_1fr]` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| padding-inline-start | `8px` | `calc(var(--spacing) * 2)` | `ps-2` |
| padding-inline-end | `16px` | `calc(var(--spacing) * 4)` | `pe-4` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-side=none]) | min-width | `calc(var(--anchor-width) + 20px)` | `in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)]` |
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ContextMenuRadioItem` → (no data-slot) · `<ContextMenuPrimitive.RadioItemIndicator>` (line 214)

#### base

```txt
col-start-1 -ms-0.5
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `1` | `1` | `col-start-1` |
| margin-inline-start | `-2px` | `calc(var(--spacing) * -0.5)` | `-ms-0.5` |

### `ContextMenuRadioItem` → (no data-slot) · `<span>` (line 230)

#### base

```txt
col-start-2
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |

### `ContextMenuGroupLabel` → `[data-slot=context-menu-label]` · `<ContextMenuPrimitive.GroupLabel>` (line 243)

Attributes: `data-inset={inset}`

#### base

```txt
px-2 py-1.5 font-medium text-muted-foreground text-xs data-inset:ps-9 sm:data-inset:ps-8
```

**element — default**

Key metrics: padding-inline `8px` · padding-block `6px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `6px` | `calc(var(--spacing) * 1.5)` | `py-1.5` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-inset] | padding-inline-start | `32px` | `sm:data-inset:ps-8` |

### `ContextMenuSeparator` → `[data-slot=context-menu-separator]` · `<ContextMenuPrimitive.Separator>` (line 260)

#### base

```txt
mx-2 my-1 h-px bg-border
```

**element — default**

Key metrics: height `1px` · background-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `8px` | `calc(var(--spacing) * 2)` | `mx-2` |
| margin-block | `4px` | `calc(var(--spacing) * 1)` | `my-1` |
| height | `1px` | `1px` | `h-px` |
| background-color | `#00000014` / `#ffffff0f` | `border` | `bg-border` |

### `ContextMenuShortcut` → `[data-slot=context-menu-shortcut]` · `<kbd>` (line 273)

#### base

```txt
ms-auto font-medium font-sans text-muted-foreground/72 text-xs tracking-widest
```

**element — default**

Key metrics: color `#686868b8` / `#818181b8` (muted-foreground/72%) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium) · letter-spacing `0.1em (1.2px)` (tracking-widest)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-start | `auto` | `auto` | `ms-auto` |
| font-family | `ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` | `font-sans` | `font-sans` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| letter-spacing | `0.1em (1.2px)` | `tracking-widest` | `tracking-widest` |
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `text-muted-foreground/72` |

### `ContextMenuSub` → `[data-slot=context-menu-sub]` · `<ContextMenuPrimitive.SubmenuRoot>` (line 288)

### `ContextMenuSubTrigger` → `[data-slot=context-menu-sub-trigger]` · `<ContextMenuPrimitive.SubmenuTrigger>` (line 301)

Attributes: `data-inset={inset}`

#### base

```txt
flex min-h-8 items-center gap-2 rounded-sm px-2 py-1 text-base text-foreground outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-popup-open:bg-accent data-inset:ps-8 data-highlighted:text-accent-foreground data-popup-open:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&>svg:not(:last-child)]:-mx-0.5 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none
```

**element — default**

Key metrics: display `flex` · min-height `28px` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |
| open | background-color | `#0000000a` / `#ffffff0a` | `data-popup-open:bg-accent` |
| [data-inset] | padding-inline-start | `32px` | `data-inset:ps-8` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

**`>svg:not(:last-child)` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&>svg:not(:last-child)]:-mx-0.5` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `ContextMenuSubTrigger` → (no data-slot) · `<ChevronRightIcon>` (line 311)

#### base

```txt
ms-auto -me-0.5 opacity-80
```

**element — default**

Key metrics: opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-start | `auto` | `auto` | `ms-auto` |
| margin-inline-end | `-2px` | `calc(var(--spacing) * -0.5)` | `-me-0.5` |
| opacity | `80%` | `80%` | `opacity-80` |

### `ContextMenuSubPopup` → `[data-slot=context-menu-sub-content]` · `<ContextMenuPopup>` (line 330)

Attributes: `align={align}`, `alignOffset={alignOffset ?? defaultAlignOffset}`, `side=inline-end`, `sideOffset={sideOffset}`

Prop defaults: `sideOffset=0`, `align="start"`

