# Input Group

Source: `apps/ui/registry/default/ui/input-group.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/input-group.mdx`

> A flexible component for grouping inputs with addons, buttons, and other elements.

Depends on: `@/ui/input`, `@/ui/textarea`

Exports: `InputGroup`, `InputGroupAddon`, `InputGroupText`, `InputGroupInput`, `InputGroupTextarea`

## Behavior & keyboard

- Wraps an input with addons (icons, text, buttons, kbd). Clicking an addon focuses the input; focus ring is drawn on the group when the inner input is `:focus-visible`.

Docs mentions:

- ### With Keyboard Shortcut

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `inputGroupAddonVariants` (line 9)

Defaults: {"align":"inline-start"}

```txt
base: flex h-auto cursor-text select-none items-center justify-center gap-2 [&>kbd]:rounded-[calc(var(--radius)-5px)] in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4.5 sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4 [&_svg]:-mx-0.5 not-has-[button]:**:[svg:not([class*='opacity-'])]:opacity-80
align.block-end: order-last w-full justify-start px-[calc(--spacing(3)-1px)] pb-[calc(--spacing(3)-1px)] [.border-t]:pt-[calc(--spacing(3)-1px)] [[data-size=sm]+&]:px-[calc(--spacing(2.5)-1px)]
align.block-start: order-first w-full justify-start px-[calc(--spacing(3)-1px)] pt-[calc(--spacing(3)-1px)] [.border-b]:pb-[calc(--spacing(3)-1px)] [[data-size=sm]+&]:px-[calc(--spacing(2.5)-1px)]
align.inline-end: order-last pe-[calc(--spacing(3)-1px)] has-[>:last-child[data-slot=badge]]:-me-1.5 has-[>button]:-me-2 has-[>kbd:last-child]:me-[-0.35rem] [[data-size=sm]+&]:pe-[calc(--spacing(2.5)-1px)]
align.inline-start: order-first ps-[calc(--spacing(3)-1px)] has-[>:last-child[data-slot=badge]]:-ms-1.5 has-[>button]:-ms-2 has-[>kbd:last-child]:ms-[-0.35rem] [[data-size=sm]+&]:ps-[calc(--spacing(2.5)-1px)]
```

## Parts

### `InputGroup` → `[data-slot=input-group]` · `<div>` (line 35)

Attributes: `role=group`

#### base

```txt
relative inline-flex w-full min-w-0 items-center rounded-lg border border-input bg-background not-dark:bg-clip-padding text-base text-foreground shadow-xs/5 ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-has-[input:disabled,textarea:disabled]:not-has-[input:focus-visible,textarea:focus-visible]:not-has-[input[aria-invalid],textarea[aria-invalid]]:before:shadow-[0_1px_--theme(--color-black/4%)] has-[input:focus-visible,textarea:focus-visible]:has-[input[aria-invalid],textarea[aria-invalid]]:border-destructive/64 has-[input:focus-visible,textarea:focus-visible]:has-[input[aria-invalid],textarea[aria-invalid]]:ring-destructive/16 has-[textarea]:h-auto has-data-[align=block-end]:h-auto has-data-[align=block-start]:h-auto has-data-[align=block-end]:flex-col has-data-[align=block-start]:flex-col has-[input:focus-visible,textarea:focus-visible]:border-ring has-[input[aria-invalid],textarea[aria-invalid]]:border-destructive/36 has-autofill:bg-foreground/4 has-[input:disabled,textarea:disabled]:opacity-64 has-[input:disabled,textarea:disabled,input:focus-visible,textarea:focus-visible,input[aria-invalid],textarea[aria-invalid]]:shadow-none has-[input:focus-visible,textarea:focus-visible]:ring-[3px] sm:text-sm dark:bg-input/32 dark:has-autofill:bg-foreground/8 dark:has-[input[aria-invalid],textarea[aria-invalid]]:ring-destructive/24 dark:not-has-[input:disabled,textarea:disabled]:not-has-[input:focus-visible,textarea:focus-visible]:not-has-[input[aria-invalid],textarea[aria-invalid]]:before:shadow-[0_-1px_--theme(--color-white/6%)] has-data-[align=inline-start]:**:[[data-size=sm]_input]:ps-1.5 has-data-[align=inline-end]:**:[[data-size=sm]_input]:pe-1.5 *:[[data-slot=input-control],[data-slot=textarea-control]]:contents *:[[data-slot=input-control],[data-slot=textarea-control]]:before:hidden has-[[data-align=block-start],[data-align=block-end]]:**:[input]:h-auto has-data-[align=inline-start]:**:[input]:ps-2 has-data-[align=inline-end]:**:[input]:pe-2 has-data-[align=block-end]:**:[input]:pt-1.5 has-data-[align=block-start]:**:[input]:pb-1.5 **:[textarea]:min-h-20.5 **:[textarea]:resize-none **:[textarea]:py-[calc(--spacing(3)-1px)] **:[textarea]:max-sm:min-h-23.5 **:[textarea_button]:rounded-[calc(var(--radius-md)-1px)]
```

**element — default**

Key metrics: display `inline-flex` · width `100%` · min-width `0px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#ffffff` / `#ffffff07` (background) · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `has-[input:disabled,textarea:disabled,input:focus-visible,textarea:focus-visible,input[aria-invalid],textarea[aria-invalid]]:shadow-none` |
| disabled | opacity | `64%` | `has-[input:disabled,textarea:disabled]:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `has-[input:focus-visible,textarea:focus-visible]:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `has-[input:focus-visible,textarea:focus-visible]:ring-[3px]` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `has-[input[aria-invalid],textarea[aria-invalid]]:border-destructive/36` |
| invalid | box-shadow | `none` | `has-[input:disabled,textarea:disabled,input:focus-visible,textarea:focus-visible,input[aria-invalid],textarea[aria-invalid]]:shadow-none` |
| has(textarea) | height | `auto` | `has-[textarea]:h-auto` |
| has([data-align=block-end]) | height | `auto` | `has-data-[align=block-end]:h-auto` |
| has([data-align=block-end]) | flex-direction | `column` | `has-data-[align=block-end]:flex-col` |
| has([data-align=block-start]) | height | `auto` | `has-data-[align=block-start]:h-auto` |
| has([data-align=block-start]) | flex-direction | `column` | `has-data-[align=block-start]:flex-col` |
| has(:autofill) | background-color | `#2626260a` / `#f5f5f514` | `has-autofill:bg-foreground/4` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `has-[input:focus-visible,textarea:focus-visible]:has-[input[aria-invalid],textarea[aria-invalid]]:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a29` | `has-[input:focus-visible,textarea:focus-visible]:ring-[3px]` |

**element — mobile differences**: font-size `16px` · line-height `24px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-has-[input:disabled,textarea:disabled]:not-has-[input:focus-visible,textarea:focus-visible]:not-has-[input[aria-invalid],textarea[aria-invalid]]:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-has-[input:disabled,textarea:disabled]:not-has-[input:focus-visible,textarea:focus-visible]:not-has-[input[aria-invalid],textarea[aria-invalid]]:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

**`*:is([data-size=sm] input)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-align=inline-start]) | padding-inline-start | `6px` | `has-data-[align=inline-start]:**:[[data-size=sm]_input]:ps-1.5` |
| has([data-align=inline-end]) | padding-inline-end | `6px` | `has-data-[align=inline-end]:**:[[data-size=sm]_input]:pe-1.5` |

**`*:is(input)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-align=block-end]) | padding-top | `6px` | `has-data-[align=block-end]:**:[input]:pt-1.5` |
| has([data-align=block-start]) | padding-bottom | `6px` | `has-data-[align=block-start]:**:[input]:pb-1.5` |
| has([data-align=inline-start]) | padding-inline-start | `8px` | `has-data-[align=inline-start]:**:[input]:ps-2` |
| has([data-align=inline-end]) | padding-inline-end | `8px` | `has-data-[align=inline-end]:**:[input]:pe-2` |
| has([data-align=block-start],[data-align=block-end]) | height | `auto` | `has-[[data-align=block-start],[data-align=block-end]]:**:[input]:h-auto` |

**`*:is(textarea button)` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `**:[textarea_button]:rounded-[calc(var(--radius-md)-1px)]` |

**`*:is(textarea)` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `82px` | `calc(var(--spacing) * 20.5)` | `**:[textarea]:min-h-20.5` |
| resize | `none` | `none` | `**:[textarea]:resize-none` |
| padding-block | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `**:[textarea]:py-[calc(--spacing(3)-1px)]` |

**`*:is(textarea)` — mobile differences**: min-height `94px`

**`> *:is([data-slot=input-control],[data-slot=textarea-control])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `contents` | `contents` | `*:[[data-slot=input-control],[data-slot=textarea-control]]:contents` |

**`> *:is([data-slot=input-control],[data-slot=textarea-control])::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:[[data-slot=input-control],[data-slot=textarea-control]]:before:hidden` |
| display | `none` | `none` | `*:[[data-slot=input-control],[data-slot=textarea-control]]:before:hidden` |

### `InputGroupAddon` → `[data-slot=input-group-addon]` · `<div>` (line 54)

Attributes: `data-align={align}`

Prop defaults: `align="inline-start"`

#### base (cva defaults: align=inline-start)

```txt
flex h-auto cursor-text select-none items-center justify-center gap-2 [&>kbd]:rounded-[calc(var(--radius)-5px)] in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4.5 sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4 [&_svg]:-mx-0.5 not-has-[button]:**:[svg:not([class*='opacity-'])]:opacity-80 order-first ps-[calc(--spacing(3)-1px)] has-[>:last-child[data-slot=badge]]:-ms-1.5 has-[>button]:-ms-2 has-[>kbd:last-child]:ms-[-0.35rem] [[data-size=sm]+&]:ps-[calc(--spacing(2.5)-1px)]
```

**element — default**

Key metrics: display `flex` · height `auto` · padding-inline-start `11px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| order | `-9999` | `-9999` | `order-first` |
| display | `flex` | `flex` | `flex` |
| height | `auto` | `auto` | `h-auto` |
| cursor | `text` | `text` | `cursor-text` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding-inline-start | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `ps-[calc(--spacing(3)-1px)]` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>:last-child[data-slot=badge]) | margin-inline-start | `-6px` | `has-[>:last-child[data-slot=badge]]:-ms-1.5` |
| has(>button) | margin-inline-start | `-8px` | `has-[>button]:-ms-2` |
| has(>kbd:last-child) | margin-inline-start | `-5.6px` | `has-[>kbd:last-child]:ms-[-0.35rem]` |
| prev([data-size=sm]) | padding-inline-start | `9px` | `[[data-size=sm]+&]:ps-[calc(--spacing(2.5)-1px)]` |

**`*:is(svg:not([class*='opacity-']))` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `not-has-[button]:**:[svg:not([class*='opacity-'])]:opacity-80` |

**`*:is(svg:not([class*='opacity-']))` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(button) | opacity | `(unset)` | `(removed)` |

**`>kbd` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `5px` | `calc(var(--radius) - 5px)` | `[&>kbd]:rounded-[calc(var(--radius)-5px)]` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-mx-0.5` |

**`svg:not([class*='size-'])` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | width | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | height | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |

#### inputGroupAddonVariants: align=block-end

```txt
order-last w-full justify-start px-[calc(--spacing(3)-1px)] pb-[calc(--spacing(3)-1px)] [.border-t]:pt-[calc(--spacing(3)-1px)] [[data-size=sm]+&]:px-[calc(--spacing(2.5)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| order | `9999` | `9999` | `order-last` |
| justify-content | `flex-start` | `flex-start` | `justify-start` |
| padding-inline-start | `(unset)` | `` | `(removed)` |
| width | `100%` | `100%` | `w-full` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| padding-bottom | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `pb-[calc(--spacing(3)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| .border-t | padding-top | `11px` | `[.border-t]:pt-[calc(--spacing(3)-1px)]` |
| prev([data-size=sm]) | padding-inline | `9px` | `[[data-size=sm]+&]:px-[calc(--spacing(2.5)-1px)]` |

**`*:is(svg:not([class*='opacity-']))` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(button) | opacity | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | width | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | height | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |

#### inputGroupAddonVariants: align=block-start

```txt
order-first w-full justify-start px-[calc(--spacing(3)-1px)] pt-[calc(--spacing(3)-1px)] [.border-b]:pb-[calc(--spacing(3)-1px)] [[data-size=sm]+&]:px-[calc(--spacing(2.5)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| justify-content | `flex-start` | `flex-start` | `justify-start` |
| padding-inline-start | `(unset)` | `` | `(removed)` |
| width | `100%` | `100%` | `w-full` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| padding-top | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `pt-[calc(--spacing(3)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| .border-b | padding-bottom | `11px` | `[.border-b]:pb-[calc(--spacing(3)-1px)]` |
| prev([data-size=sm]) | padding-inline | `9px` | `[[data-size=sm]+&]:px-[calc(--spacing(2.5)-1px)]` |

**`*:is(svg:not([class*='opacity-']))` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(button) | opacity | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | width | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | height | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |

#### inputGroupAddonVariants: align=inline-end

```txt
order-last pe-[calc(--spacing(3)-1px)] has-[>:last-child[data-slot=badge]]:-me-1.5 has-[>button]:-me-2 has-[>kbd:last-child]:me-[-0.35rem] [[data-size=sm]+&]:pe-[calc(--spacing(2.5)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| order | `9999` | `9999` | `order-last` |
| padding-inline-start | `(unset)` | `` | `(removed)` |
| padding-inline-end | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `pe-[calc(--spacing(3)-1px)]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>:last-child[data-slot=badge]) | margin-inline-end | `-6px` | `has-[>:last-child[data-slot=badge]]:-me-1.5` |
| has(>button) | margin-inline-end | `-8px` | `has-[>button]:-me-2` |
| has(>kbd:last-child) | margin-inline-end | `-5.6px` | `has-[>kbd:last-child]:me-[-0.35rem]` |
| prev([data-size=sm]) | padding-inline-end | `9px` | `[[data-size=sm]+&]:pe-[calc(--spacing(2.5)-1px)]` |

**`*:is(svg:not([class*='opacity-']))` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(button) | opacity | `(unset)` | `(removed)` |

**`svg:not([class*='size-'])` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | width | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | height | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |

### `InputGroupText` → (no data-slot) · `<span>` (line 85)

#### base

```txt
flex items-center gap-2 truncate text-muted-foreground in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4.5 sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5
```

**element — default**

Key metrics: display `flex` · gap `8px` · color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| overflow | `hidden` | `hidden` | `truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `truncate` |
| white-space | `nowrap` | `nowrap` | `truncate` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-mx-0.5` |

**`svg:not([class*='size-'])` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | width | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |
| in(:where(*:is([data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])))) | height | `16px` | `sm:in-[[data-slot=input-group]:has([data-slot=input-control],[data-slot=textarea-control])]:[&_svg:not([class*='size-'])]:size-4` |

### `InputGroupInput`

No own classes (re-export / wrapper of the primitive).

### `InputGroupTextarea`

No own classes (re-export / wrapper of the primitive).

