# Sheet

Source: `apps/ui/registry/default/ui/sheet.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/sheet.mdx` · Base UI: https://base-ui.com/react/components/dialog#api-reference

> A flyout that opens from the side of the screen, based on the dialog component.

Depends on: `@base-ui/react/dialog`, `@base-ui/react/merge-props`, `@base-ui/react/use-render`, `lucide-react`, `@/ui/button`, `@/ui/scroll-area`

Exports: `SheetTrigger`, `SheetClose`, `SheetBackdrop`, `SheetViewport`, `SheetPopup`, `SheetHeader`, `SheetFooter`, `SheetTitle`, `SheetDescription`, `SheetPanel`

## Behavior & keyboard

- Dialog variant sliding in from a side (`side` prop: top/right/bottom/left). Same focus/escape behavior as Dialog. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `SheetTrigger` → `[data-slot=sheet-trigger]` · `<SheetPrimitive.Trigger>` (line 19)

### `SheetClose` → `[data-slot=sheet-close]` · `<SheetPrimitive.Close>` (line 25)

### `SheetBackdrop` → `[data-slot=sheet-backdrop]` · `<SheetPrimitive.Backdrop>` (line 33)

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

### `SheetViewport` → `[data-slot=sheet-viewport]` · `<SheetPrimitive.Viewport>` (line 54)

Prop defaults: `variant="default"`

#### base

```txt
fixed inset-0 z-50 grid
```

**element — default**

Key metrics: display `grid` · z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `50` | `50` | `z-50` |
| display | `grid` | `grid` | `grid` |

#### when `side === "bottom"`

```txt
grid grid-rows-[1fr_auto] pt-12
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-template-rows | `1fr auto` | `1fr auto` | `grid-rows-[1fr_auto]` |
| padding-top | `48px` | `calc(var(--spacing) * 12)` | `pt-12` |

#### when `side === "top"`

```txt
grid grid-rows-[auto_1fr] pb-12
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-template-rows | `auto 1fr` | `auto 1fr` | `grid-rows-[auto_1fr]` |
| padding-bottom | `48px` | `calc(var(--spacing) * 12)` | `pb-12` |

#### when `side === "left"`

```txt
flex justify-start
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| justify-content | `flex-start` | `flex-start` | `justify-start` |

#### when `side === "right"`

```txt
flex justify-end
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| justify-content | `flex-end` | `flex-end` | `justify-end` |

#### when `variant === "inset"`

```txt
sm:p-4
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding | `16px` | `calc(var(--spacing) * 4)` | `sm:p-4` |

**element — mobile differences**: padding `(unset)`

### `SheetPopup` → `[data-slot=sheet-popup]` · `<SheetPrimitive.Popup>` (line 90)

Prop defaults: `showCloseButton=true`, `side="right"`, `variant="default"`

#### base

```txt
relative flex max-h-full min-h-0 w-full min-w-0 flex-col bg-popover not-dark:bg-clip-padding text-popover-foreground shadow-lg/5 transition-[opacity,translate] duration-200 ease-in-out will-change-transform before:pointer-events-none before:absolute before:inset-0 before:shadow-[0_1px_--theme(--color-black/4%)] data-ending-style:opacity-0 data-starting-style:opacity-0 max-sm:before:hidden dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · min-height `0px` · max-height `100%` · width `100%` · min-width `0px` · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · transition-property `opacity, translate` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| max-height | `100%` | `100%` | `max-h-full` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `opacity, translate` | `opacity,translate` | `transition-[opacity,translate]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |
| will-change | `transform` | `transform` | `will-change-transform` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — mobile differences**: display `none`

#### when `side === "bottom"`

```txt
row-start-2 border-t data-ending-style:translate-y-8 data-starting-style:translate-y-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-row-start | `2` | `2` | `row-start-2` |
| border-top-style | `solid` | `var(--tw-border-style)` | `border-t` |
| border-top-width | `1px` | `1px` | `border-t` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | translate | `0 32px` | `data-ending-style:translate-y-8` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | translate | `0 32px` | `data-starting-style:translate-y-8` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — mobile differences**: display `none`

#### when `side === "top"`

```txt
border-b data-ending-style:-translate-y-8 data-starting-style:-translate-y-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-style | `solid` | `var(--tw-border-style)` | `border-b` |
| border-bottom-width | `1px` | `1px` | `border-b` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | translate | `0 -32px` | `data-ending-style:-translate-y-8` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | translate | `0 -32px` | `data-starting-style:-translate-y-8` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — mobile differences**: display `none`

#### when `side === "left"`

```txt
w-[calc(100%-(--spacing(12)))] max-w-md border-e data-ending-style:-translate-x-8 data-starting-style:-translate-x-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| max-width | `448px` | `container-md` | `max-w-md` |
| border-inline-end-style | `solid` | `var(--tw-border-style)` | `border-e` |
| border-inline-end-width | `1px` | `1px` | `border-e` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | translate | `-32px 0` | `data-ending-style:-translate-x-8` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | translate | `-32px 0` | `data-starting-style:-translate-x-8` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — mobile differences**: display `none`

#### when `side === "right"`

```txt
col-start-2 w-[calc(100%-(--spacing(12)))] max-w-md border-s data-ending-style:translate-x-8 data-starting-style:translate-x-8
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| max-width | `448px` | `container-md` | `max-w-md` |
| border-inline-start-style | `solid` | `var(--tw-border-style)` | `border-s` |
| border-inline-start-width | `1px` | `1px` | `border-s` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | translate | `32px 0` | `data-ending-style:translate-x-8` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | translate | `32px 0` | `data-starting-style:translate-x-8` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — mobile differences**: display `none`

#### when `variant === "inset"`

```txt
before:hidden sm:rounded-2xl sm:border sm:before:rounded-[calc(var(--radius-2xl)-1px)] sm:**:data-[slot=sheet-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `sm:border` |
| border-style | `solid` | `var(--tw-border-style)` | `sm:border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| border-radius | `16px` | `radius-2xl` | `sm:rounded-2xl` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**element — mobile differences**: border-width `(unset)` · border-style `(unset)` · border-color `(unset)` · border-radius `(unset)`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `none` | `none` | `before:hidden` |
| border-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:before:rounded-[calc(var(--radius-2xl)-1px)]` |

**`::before` — mobile differences**: border-radius `(unset)`

**`*[data-slot="sheet-footer"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:**:data-[slot=sheet-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]` |
| border-bottom-left-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `sm:**:data-[slot=sheet-footer]:rounded-b-[calc(var(--radius-2xl)-1px)]` |

**`*[data-slot="sheet-footer"]` — mobile differences**: border-bottom-right-radius `(unset)` · border-bottom-left-radius `(unset)`

### `SheetPopup` → (no data-slot) · `<SheetPrimitive.Close>` (line 110)

Attributes: `aria-label=Close`, `render={<Button size="icon" variant="ghost" />}`

Prop defaults: `showCloseButton=true`, `side="right"`, `variant="default"`

#### base

```txt
absolute end-2 top-2
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset-inline-end | `8px` | `calc(var(--spacing) * 2)` | `end-2` |
| top | `8px` | `calc(var(--spacing) * 2)` | `top-2` |

### `SheetHeader` → `[data-slot=sheet-header]` · `useRender <div>` (line 130)

#### base

```txt
flex flex-col gap-2 p-6 in-[[data-slot=sheet-popup]:has([data-slot=sheet-panel])]:pb-3 max-sm:pb-4
```

**element — default**

Key metrics: display `flex` · padding `24px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=sheet-popup]:has([data-slot=sheet-panel])))) | padding-bottom | `12px` | `in-[[data-slot=sheet-popup]:has([data-slot=sheet-panel])]:pb-3` |

**element — mobile differences**: padding-bottom `16px`

### `SheetFooter` → `[data-slot=sheet-footer]` · `useRender <div>` (line 153)

Prop defaults: `variant="default"`

#### base

```txt
flex flex-col-reverse gap-2 px-6 sm:flex-row sm:justify-end
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

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

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

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

#### when `variant === "bare"`

```txt
in-[[data-slot=sheet-popup]:has([data-slot=sheet-panel])]:pt-3 pt-4 pb-6
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-top | `16px` | `calc(var(--spacing) * 4)` | `pt-4` |
| padding-bottom | `24px` | `calc(var(--spacing) * 6)` | `pb-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=sheet-popup]:has([data-slot=sheet-panel])))) | padding-top | `12px` | `in-[[data-slot=sheet-popup]:has([data-slot=sheet-panel])]:pt-3` |

**element — mobile differences**: flex-direction `column-reverse` · justify-content `(unset)`

### `SheetTitle` → `[data-slot=sheet-title]` · `<SheetPrimitive.Title>` (line 176)

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

### `SheetDescription` → `[data-slot=sheet-description]` · `<SheetPrimitive.Description>` (line 189)

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

### `SheetPanel` → `[data-slot=sheet-panel]` · `useRender <div>` (line 205)

Prop defaults: `scrollFade=true`

#### base

```txt
p-6 in-[[data-slot=sheet-popup]:has([data-slot=sheet-header])]:pt-1 in-[[data-slot=sheet-popup]:has([data-slot=sheet-footer]:not(.border-t))]:pb-1
```

**element — default**

Key metrics: padding `24px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=sheet-popup]:has([data-slot=sheet-header])))) | padding-top | `4px` | `in-[[data-slot=sheet-popup]:has([data-slot=sheet-header])]:pt-1` |
| in(:where(*:is([data-slot=sheet-popup]:has([data-slot=sheet-footer]:not(.border-t))))) | padding-bottom | `4px` | `in-[[data-slot=sheet-popup]:has([data-slot=sheet-footer]:not(.border-t))]:pb-1` |

