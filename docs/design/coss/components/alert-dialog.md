# Alert Dialog

Source: `apps/ui/registry/default/ui/alert-dialog.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/alert-dialog.mdx` · Base UI: https://base-ui.com/react/components/alert-dialog#api-reference

> A dialog that requires user response to proceed.

Depends on: `@base-ui/react/alert-dialog`

Exports: `AlertDialogTrigger`, `AlertDialogBackdrop`, `AlertDialogViewport`, `AlertDialogPopup`, `AlertDialogHeader`, `AlertDialogFooter`, `AlertDialogTitle`, `AlertDialogDescription`, `AlertDialogClose`

## Behavior & keyboard

- Modal; focus is trapped inside the popup and moved to the first focusable element on open; returns to trigger on close.
- `Escape` closes; clicking the backdrop does **not** close (alert dialog semantics, `role=alertdialog`).
- Backdrop + popup fade/scale via data-starting-style / data-ending-style. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `AlertDialogTrigger` → `[data-slot=alert-dialog-trigger]` · `<AlertDialogPrimitive.Trigger>` (line 20)

### `AlertDialogBackdrop` → `[data-slot=alert-dialog-backdrop]` · `<AlertDialogPrimitive.Backdrop>` (line 29)

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

### `AlertDialogViewport` → `[data-slot=alert-dialog-viewport]` · `<AlertDialogPrimitive.Viewport>` (line 45)

#### base

```txt
fixed inset-0 z-50 grid grid-rows-[1fr_auto_3fr] justify-items-center p-4
```

**element — default**

Key metrics: display `grid` · padding `16px` · z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| display | `grid` | `grid` | `grid` |
| grid-template-rows | `1fr auto 3fr` | `1fr auto 3fr` | `grid-rows-[1fr_auto_3fr]` |
| justify-items | `center` | `center` | `justify-items-center` |
| padding | `16px` | `calc(var(--spacing) * 4)` | `p-4` |

### `AlertDialogPopup` → (no data-slot) · `<AlertDialogViewport>` (line 68)

Prop defaults: `bottomStickOnMobile=true`

#### when `bottomStickOnMobile`

```txt
max-sm:grid-rows-[1fr_auto] max-sm:p-0 max-sm:pt-12
```

**element — mobile differences**: grid-template-rows `1fr auto` · padding `0px` · padding-top `48px`

### `AlertDialogPopup` → `[data-slot=alert-dialog-popup]` · `<AlertDialogPrimitive.Popup>` (line 74)

Prop defaults: `bottomStickOnMobile=true`

#### base

```txt
relative row-start-2 flex max-h-full min-h-0 w-full min-w-0 max-w-lg origin-center flex-col rounded-2xl border bg-popover not-dark:bg-clip-padding text-popover-foreground opacity-[calc(1-var(--nested-dialogs))] shadow-lg/5 transition-[scale,opacity,translate] duration-200 ease-in-out will-change-transform before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-2xl)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] data-ending-style:opacity-0 data-starting-style:opacity-0 sm:scale-[calc(1-0.1*var(--nested-dialogs))] sm:data-ending-style:scale-98 sm:data-starting-style:scale-98 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · min-height `0px` · max-height `100%` · width `100%` · min-width `0px` · max-width `512px` (container-lg) · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `16px` (radius-2xl) · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · opacity `calc(1 - var(--nested-dialogs))` · transition-property `scale, opacity, translate` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| grid-row-start | `2` | `2` | `row-start-2` |
| display | `flex` | `flex` | `flex` |
| max-height | `100%` | `100%` | `max-h-full` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| width | `100%` | `100%` | `w-full` |
| max-width | `512px` | `container-lg` | `max-w-lg` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| transform-origin | `center` | `center` | `origin-center` |
| flex-direction | `column` | `column` | `flex-col` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| opacity | `calc(1 - var(--nested-dialogs))` | `calc(1 - var(--nested-dialogs))` | `opacity-[calc(1-var(--nested-dialogs))]` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `scale, opacity, translate` | `scale,opacity,translate` | `transition-[scale,opacity,translate]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |
| will-change | `transform` | `transform` | `will-change-transform` |
| scale | `calc(1 - 0.1 * var(--nested-dialogs))` | `calc(1 - 0.1 * var(--nested-dialogs))` | `sm:scale-[calc(1-0.1*var(--nested-dialogs))]` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| ending-style (exit to) | scale | `98% 98%` | `sm:data-ending-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `sm:data-starting-style:scale-98` |

**element — mobile differences**: scale `(unset)`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-[calc(var(--radius-2xl)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

#### when `bottomStickOnMobile`

```txt
max-sm:max-w-none max-sm:origin-bottom max-sm:rounded-none max-sm:border-x-0 max-sm:border-t max-sm:border-b-0 max-sm:data-ending-style:translate-y-4 max-sm:data-starting-style:translate-y-4 max-sm:before:hidden max-sm:before:rounded-none
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| ending-style (exit to) | scale | `98% 98%` | `sm:data-ending-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `sm:data-starting-style:scale-98` |

**element — mobile differences**: max-width `none` · transform-origin `bottom` · border-radius `0` · scale `(unset)` · border-inline-style `solid` · border-inline-width `0px` · border-top-style `solid` · border-top-width `1px` · border-bottom-style `solid` · border-bottom-width `0px`

**`::before` — mobile differences**: border-radius `0` · display `none`

### `AlertDialogHeader` → `[data-slot=alert-dialog-header]` · `<div>` (line 94)

#### base

```txt
flex flex-col gap-2 p-6 text-center max-sm:pb-4 sm:text-left
```

**element — default**

Key metrics: display `flex` · padding `24px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |
| text-align | `left` | `left` | `sm:text-left` |

**element — mobile differences**: text-align `center` · padding-bottom `16px`

### `AlertDialogFooter` → `[data-slot=alert-dialog-footer]` · `<div>` (line 113)

Prop defaults: `variant="default"`

#### base

```txt
flex flex-col-reverse gap-2 px-6 sm:flex-row sm:justify-end sm:rounded-b-[calc(var(--radius-2xl)-1px)]
```

**element — default**

Key metrics: display `flex` · padding-inline `24px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `row` | `row` | `sm:flex-row` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding-inline | `24px` | `calc(var(--spacing) * 6)` | `px-6` |
| justify-content | `flex-end` | `flex-end` | `sm:justify-end` |
| border-bottom-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-bottom-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:rounded-b-[calc(var(--radius-2xl)-1px)]` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)` · border-bottom-right-radius `(unset)` · border-bottom-left-radius `(unset)`

#### when `variant === "default"`

```txt
border-t bg-muted/72 py-4
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-top-style | `solid` | `var(--tw-border-style)` | `border-t` |
| border-top-width | `1px` | `1px` | `border-t` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `bg-muted/72` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)` · border-bottom-right-radius `(unset)` · border-bottom-left-radius `(unset)`

#### when `variant === "bare"`

```txt
pb-6
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-bottom | `24px` | `calc(var(--spacing) * 6)` | `pb-6` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)` · border-bottom-right-radius `(unset)` · border-bottom-left-radius `(unset)`

### `AlertDialogTitle` → `[data-slot=alert-dialog-title]` · `<AlertDialogPrimitive.Title>` (line 131)

#### base

```txt
font-bold font-heading text-xl leading-none
```

**element — default**

Key metrics: font-size `20px` (text-xl) · line-height `20px` · font-weight `700` (font-weight-bold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-family | `ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` | `font-sans` | `font-heading` |
| font-size | `20px` | `text-xl` | `text-xl` |
| line-height | `20px` | `1` | `leading-none` |
| font-weight | `700` | `font-weight-bold` | `font-bold` |

### `AlertDialogDescription` → `[data-slot=alert-dialog-description]` · `<AlertDialogPrimitive.Description>` (line 144)

#### base

```txt
text-muted-foreground text-sm
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `AlertDialogClose` → `[data-slot=alert-dialog-close]` · `<AlertDialogPrimitive.Close>` (line 156)

