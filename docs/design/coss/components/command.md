# Command

Source: `apps/ui/registry/default/ui/command.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/command.mdx` · Base UI: https://base-ui.com/react/components/autocomplete#api-reference

> A command palette component built with Dialog and Autocomplete for searching and executing commands.

Depends on: `@base-ui/react/dialog`, `lucide-react`, `@/ui/autocomplete`

Exports: `CommandDialogTrigger`, `CommandDialogBackdrop`, `CommandDialogViewport`, `CommandDialogPopup`, `Command`, `CommandInput`, `CommandList`, `CommandEmpty`, `CommandPanel`, `CommandGroup`, `CommandGroupLabel`, `CommandItem`, `CommandSeparator`, `CommandShortcut`, `CommandFooter`

## Behavior & keyboard

- Command palette (dialog + autocomplete list). `⌘K`/`Ctrl+K` typically opens (app-defined); `ArrowUp/Down` highlight, `Enter` runs, `Escape` closes. Footer shows keyboard hints with `Kbd`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

Docs mentions:

- CommandShortcut,
- ### CommandShortcut
- Displays keyboard shortcuts in a styled span element.
- A footer section for displaying hints or additional keyboard shortcuts. Renders as a styled `div` with padding and border.
- ### With Keyboard Shortcut
- You can add a keyboard shortcut to open the command palette:
- const down = (e: KeyboardEvent) => {
- if (e.key === "k" && (e.metaKey || e.ctrlKey)) {

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `CommandDialogTrigger` → `[data-slot=command-dialog-trigger]` · `<CommandDialogPrimitive.Trigger>` (line 32)

### `CommandDialogBackdrop` → `[data-slot=command-dialog-backdrop]` · `<CommandDialogPrimitive.Backdrop>` (line 44)

#### base

```txt
fixed inset-0 z-50 bg-black/32 backdrop-blur-sm transition-all duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0 supports-[-webkit-touch-callout:none]:absolute
```

**element — default**

Key metrics: background-color `#00000052` (color-black/32%) · z-index `50` · transition-property `all` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| background-color | `#00000052` | `color-black/32%` | `bg-black/32` |
| -webkit-backdrop-filter | `blur(8px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-sm` |
| backdrop-filter | `blur(8px)` | `var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-graysca` | `backdrop-blur-sm` |
| transition-property | `all` | `all` | `transition-all` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-all` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

### `CommandDialogViewport` → `[data-slot=command-dialog-viewport]` · `<CommandDialogPrimitive.Viewport>` (line 60)

#### base

```txt
fixed inset-0 z-50 flex flex-col items-center px-4 py-[max(--spacing(4),4vh)] sm:py-[10vh]
```

**element — default**

Key metrics: display `flex` · padding-inline `16px` · padding-block `10vh` · z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `center` | `center` | `items-center` |
| padding-inline | `16px` | `calc(var(--spacing) * 4)` | `px-4` |
| padding-block | `10vh` | `10vh` | `sm:py-[10vh]` |

**element — mobile differences**: padding-block `max(16px, 4vh)`

### `CommandDialogPopup` → `[data-slot=command-dialog-popup]` · `<CommandDialogPrimitive.Popup>` (line 83)

#### base

```txt
relative row-start-2 flex max-h-105 min-h-0 w-full min-w-0 max-w-xl -translate-y-[calc(1.25rem*var(--nested-dialogs))] scale-[calc(1-0.1*var(--nested-dialogs))] flex-col rounded-2xl border bg-popover not-dark:bg-clip-padding text-popover-foreground opacity-[calc(1-0.1*var(--nested-dialogs))] shadow-lg/5 outline-none transition-[scale,opacity,translate] duration-200 ease-in-out will-change-transform before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-2xl)-1px)] before:bg-muted/72 before:shadow-[0_1px_--theme(--color-black/4%)] data-nested:data-ending-style:translate-y-8 data-nested:data-starting-style:translate-y-8 data-nested-dialog-open:origin-top data-ending-style:scale-98 data-starting-style:scale-98 data-ending-style:opacity-0 data-starting-style:opacity-0 **:data-[slot=scroll-area-viewport]:data-has-overflow-y:pe-1 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · min-height `0px` · max-height `420px` · width `100%` · min-width `0px` · max-width `576px` (container-xl) · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `16px` (radius-2xl) · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · opacity `calc(1 - 0.1 * var(--nested-dialogs))` · transition-property `scale, opacity, translate` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| grid-row-start | `2` | `2` | `row-start-2` |
| display | `flex` | `flex` | `flex` |
| max-height | `420px` | `calc(var(--spacing) * 105)` | `max-h-105` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| width | `100%` | `100%` | `w-full` |
| max-width | `576px` | `container-xl` | `max-w-xl` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| translate | `0 calc(calc(20px * var(--nested-dialogs)) * -1)` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-y-[calc(1.25rem*var(--nested-dialogs))]` |
| scale | `calc(1 - 0.1 * var(--nested-dialogs))` | `calc(1 - 0.1 * var(--nested-dialogs))` | `scale-[calc(1-0.1*var(--nested-dialogs))]` |
| flex-direction | `column` | `column` | `flex-col` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| opacity | `calc(1 - 0.1 * var(--nested-dialogs))` | `calc(1 - 0.1 * var(--nested-dialogs))` | `opacity-[calc(1-0.1*var(--nested-dialogs))]` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `scale, opacity, translate` | `scale,opacity,translate` | `transition-[scale,opacity,translate]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |
| will-change | `transform` | `transform` | `will-change-transform` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | scale | `98% 98%` | `data-ending-style:scale-98` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `data-starting-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| [data-nested-dialog-open] | transform-origin | `top` | `data-nested-dialog-open:origin-top` |
| [data-nested] + ending-style (exit to) | translate | `0 32px` | `data-nested:data-ending-style:translate-y-8` |
| [data-nested] + ending-style (exit to) | scale | `98% 98%` | `data-ending-style:scale-98` |
| [data-nested] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-nested] + starting-style (enter from) | translate | `0 32px` | `data-nested:data-starting-style:translate-y-8` |
| [data-nested] + starting-style (enter from) | scale | `98% 98%` | `data-starting-style:scale-98` |
| [data-nested] + starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-[calc(var(--radius-2xl)-1px)]` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `before:bg-muted/72` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`*[data-slot="scroll-area-viewport"][data-has-overflow-y]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-end | `4px` | `calc(var(--spacing) * 1)` | `**:data-[slot=scroll-area-viewport]:data-has-overflow-y:pe-1` |

### `Command`

No own classes (re-export / wrapper of the primitive); prop defaults: `autoHighlight="always"`, `keepHighlight=true`.

### `CommandInput` → (no data-slot) · `<div>` (line 120)

Prop defaults: `placeholder=undefined`

#### base

```txt
px-2.5 py-1.5
```

**element — default**

Key metrics: padding-inline `10px` · padding-block `6px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `10px` | `calc(var(--spacing) * 2.5)` | `px-2.5` |
| padding-block | `6px` | `calc(var(--spacing) * 1.5)` | `py-1.5` |

### `CommandInput` → (no data-slot) · `<AutocompleteInput>` (line 121)

Attributes: `autoFocus=true`, `placeholder={placeholder}`, `size=lg`, `startAddon={<SearchIcon />}`

Prop defaults: `placeholder=undefined`

#### base

```txt
border-transparent! bg-transparent! shadow-none before:hidden has-focus-visible:ring-0
```

**element — default**

Key metrics: border-color `#00000000` · background-color `#00000000` · box-shadow `none`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#00000000` | `transparent` | `border-transparent!` |
| box-shadow | `none` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-none` |
| background-color | `#00000000` | `transparent` | `bg-transparent!` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 0px currentcolor` | `has-focus-visible:ring-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:hidden` |
| display | `none` | `none` | `before:hidden` |

### `CommandList` → `[data-slot=command-list]` · `<AutocompleteList>` (line 141)

#### base

```txt
not-empty:scroll-py-2 not-empty:p-2
```

**element — default**

Key metrics: padding `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| scroll-padding-block | `8px` | `calc(var(--spacing) * 2)` | `not-empty:scroll-py-2` |
| padding | `8px` | `calc(var(--spacing) * 2)` | `not-empty:p-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | scroll-padding-block | `(unset)` | `(removed)` |
| :empty | padding | `(unset)` | `(removed)` |

### `CommandEmpty` → `[data-slot=command-empty]` · `<AutocompleteEmpty>` (line 154)

#### base

```txt
not-empty:py-6
```

**element — default**

Key metrics: padding-block `24px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-block | `24px` | `calc(var(--spacing) * 6)` | `not-empty:py-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | padding-block | `(unset)` | `(removed)` |

### `CommandPanel` → (no data-slot) · `<div>` (line 167)

#### base

```txt
relative -mx-px not-has-[+[data-slot=command-footer]]:-mb-px min-h-0 rounded-t-xl not-has-[+[data-slot=command-footer]]:rounded-b-2xl border border-b-0 bg-popover bg-clip-padding shadow-xs/5 [clip-path:inset(0_1px)] not-has-[+[data-slot=command-footer]]:[clip-path:inset(0_1px_1px_1px_round_0_0_calc(var(--radius-2xl)-1px)_calc(var(--radius-2xl)-1px))] before:pointer-events-none before:absolute before:inset-0 before:rounded-t-[calc(var(--radius-xl)-1px)] **:data-[slot=scroll-area-scrollbar]:mt-2
```

**element — default**

Key metrics: min-height `0px` · border-width `1px` · border-bottom-width `0px` · border-color `#00000014` / `#ffffff0f` (border) · background-color `#ffffff` / `#1d1d1d` (popover) · box-shadow `0 1px 2px 0 #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| margin-inline | `-1px` | `-1px` | `-mx-px` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| border-top-left-radius | `14px` | `calc(var(--radius) + 4px)` | `rounded-t-xl` |
| border-top-right-radius | `14px` | `calc(var(--radius) + 4px)` | `rounded-t-xl` |
| border-bottom-style | `solid` | `var(--tw-border-style)` | `border-b-0` |
| border-bottom-width | `0px` | `0px` | `border-b-0` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| background-clip | `padding-box` | `padding-box` | `bg-clip-padding` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| clip-path | `inset(0 1px 1px 1px round 0 0 15px 15px)` | `inset(0 1px 1px 1px round 0 0 calc(var(--radius-2xl) - 1px) calc(var(--radius-2xl) - 1px))` | `not-has-[+[data-slot=command-footer]]:[clip-path:inset(0_1px_1px_1px_round_0_0_calc(var(--radius-2xl)-1px)_calc(var(--radius-2xl)-1px))]` |
| margin-bottom | `-1px` | `-1px` | `not-has-[+[data-slot=command-footer]]:-mb-px` |
| border-bottom-right-radius | `16px` | `radius-2xl` | `not-has-[+[data-slot=command-footer]]:rounded-b-2xl` |
| border-bottom-left-radius | `16px` | `radius-2xl` | `not-has-[+[data-slot=command-footer]]:rounded-b-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-slot=command-footer]) | clip-path | `inset(0 1px)` | `[clip-path:inset(0_1px)]` |
| has(+[data-slot=command-footer]) | margin-bottom | `(unset)` | `(removed)` |
| has(+[data-slot=command-footer]) | border-bottom-right-radius | `(unset)` | `(removed)` |
| has(+[data-slot=command-footer]) | border-bottom-left-radius | `(unset)` | `(removed)` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-t-[calc(var(--radius-xl)-1px)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-top-left-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `before:rounded-t-[calc(var(--radius-xl)-1px)]` |
| border-top-right-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `before:rounded-t-[calc(var(--radius-xl)-1px)]` |

**`*[data-slot="scroll-area-scrollbar"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-top | `8px` | `calc(var(--spacing) * 2)` | `**:data-[slot=scroll-area-scrollbar]:mt-2` |

### `CommandGroup` → `[data-slot=command-group]` · `<AutocompleteGroup>` (line 182)

### `CommandGroupLabel` → `[data-slot=command-group-label]` · `<AutocompleteGroupLabel>` (line 195)

### `CommandItem` → `[data-slot=command-item]` · `<AutocompleteItem>` (line 210)

#### base

```txt
py-1.5
```

**element — default**

Key metrics: padding-block `6px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-block | `6px` | `calc(var(--spacing) * 1.5)` | `py-1.5` |

### `CommandSeparator` → `[data-slot=command-separator]` · `<AutocompleteSeparator>` (line 223)

#### base

```txt
my-2
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-block | `8px` | `calc(var(--spacing) * 2)` | `my-2` |

### `CommandShortcut` → `[data-slot=command-shortcut]` · `<kbd>` (line 236)

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

### `CommandFooter` → `[data-slot=command-footer]` · `<div>` (line 252)

#### base

```txt
flex items-center justify-between gap-2 rounded-b-[calc(var(--radius-2xl)-1px)] border-t px-5 py-3 text-muted-foreground text-xs
```

**element — default**

Key metrics: display `flex` · padding-inline `20px` · padding-block `12px` · gap `8px` · border-top-width `1px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-bottom-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-bottom-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-top-style | `solid` | `var(--tw-border-style)` | `border-t` |
| border-top-width | `1px` | `1px` | `border-t` |
| padding-inline | `20px` | `calc(var(--spacing) * 5)` | `px-5` |
| padding-block | `12px` | `calc(var(--spacing) * 3)` | `py-3` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

