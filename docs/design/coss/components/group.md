# Group

Source: `apps/ui/registry/default/ui/group.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/group.mdx`

> A component for visually grouping a series of controls.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`, `@/ui/separator`

Exports: `Group`, `GroupText`, `GroupSeparator`

## Behavior & keyboard

- Visual grouping of buttons/inputs with shared borders (`role=group`); each child remains its own tab stop.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `groupVariants` (line 10)

Defaults: {"orientation":"horizontal"}

```txt
base: flex w-fit *:focus-visible:z-1 has-[>[data-slot=group]]:gap-2 *:has-focus-visible:z-1 dark:*:[[data-slot=separator]:has(~button:hover):not(:has(~[data-slot=separator]~[data-slot]:hover)),[data-slot=separator]:has(~[data-slot][data-pressed]):not(:has(~[data-slot=separator]~[data-slot][data-pressed]))]:before:bg-input/64 dark:*:[button:hover~[data-slot=separator]:not([data-slot]:hover~[data-slot=separator]~[data-slot=separator]),[data-slot][data-pressed]~[data-slot=separator]:not([data-slot][data-pressed]~[data-slot=separator]~[data-slot=separator])]:before:bg-input/64
orientation.horizontal: *:pointer-coarse:after:min-w-auto *:data-slot:has-[~[data-slot]]:rounded-e-none *:data-slot:has-[~[data-slot]]:border-e-0 *:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-end-[0.5px] *:data-slot:has-[~[data-slot]]:before:rounded-e-none *:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-start-[0.5px] *:[[data-slot]~[data-slot]]:rounded-s-none *:[[data-slot]~[data-slot]]:border-s-0 *:[[data-slot]~[data-slot]]:before:rounded-s-none
orientation.vertical: flex-col *:pointer-coarse:after:min-h-auto *:data-slot:has-[~[data-slot]]:rounded-b-none *:data-slot:has-[~[data-slot]]:border-b-0 *:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-bottom-[0.5px] *:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:hidden *:data-slot:has-[~[data-slot]]:before:rounded-b-none dark:*:last:before:hidden dark:*:first:before:block *:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-top-[0.5px] *:[[data-slot]~[data-slot]]:rounded-t-none *:[[data-slot]~[data-slot]]:border-t-0 *:[[data-slot]~[data-slot]]:before:rounded-t-none
```

## Parts

### `Group` → `[data-slot=group]` · `<div>` (line 38)

Attributes: `data-orientation={orientation}`, `role=group`

#### base (cva defaults: orientation=horizontal)

```txt
flex w-fit *:focus-visible:z-1 has-[>[data-slot=group]]:gap-2 *:has-focus-visible:z-1 dark:*:[[data-slot=separator]:has(~button:hover):not(:has(~[data-slot=separator]~[data-slot]:hover)),[data-slot=separator]:has(~[data-slot][data-pressed]):not(:has(~[data-slot=separator]~[data-slot][data-pressed]))]:before:bg-input/64 dark:*:[button:hover~[data-slot=separator]:not([data-slot]:hover~[data-slot=separator]~[data-slot=separator]),[data-slot][data-pressed]~[data-slot=separator]:not([data-slot][data-pressed]~[data-slot=separator]~[data-slot=separator])]:before:bg-input/64 *:pointer-coarse:after:min-w-auto *:data-slot:has-[~[data-slot]]:rounded-e-none *:data-slot:has-[~[data-slot]]:border-e-0 *:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-end-[0.5px] *:data-slot:has-[~[data-slot]]:before:rounded-e-none *:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-start-[0.5px] *:[[data-slot]~[data-slot]]:rounded-s-none *:[[data-slot]~[data-slot]]:border-s-0 *:[[data-slot]~[data-slot]]:before:rounded-s-none
```

**element — default**

Key metrics: display `flex` · width `fit-content`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `fit-content` | `fit-content` | `w-fit` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>[data-slot=group]) | gap | `8px` | `has-[>[data-slot=group]]:gap-2` |

**`> *::after` — mobile differences**: content `""` · min-width `auto`

**`> *:focus-visible` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `1` | `1` | `*:focus-visible:z-1` |

**`> *:has(*:focus-visible)` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `1` | `1` | `*:has-focus-visible:z-1` |

**`> *:is([data-slot]~[data-slot]:not([data-slot=separator]))::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-start-[0.5px]` |
| inset-inline-start | `-0.5px` | `calc(0.5px * -1)` | `*:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-start-[0.5px]` |

**`> *:is([data-slot]~[data-slot])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:rounded-s-none` |
| border-end-start-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:rounded-s-none` |
| border-inline-start-style | `solid` | `var(--tw-border-style)` | `*:[[data-slot]~[data-slot]]:border-s-0` |
| border-inline-start-width | `0px` | `0px` | `*:[[data-slot]~[data-slot]]:border-s-0` |

**`> *:is([data-slot]~[data-slot])::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:[[data-slot]~[data-slot]]:before:rounded-s-none` |
| border-start-start-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:before:rounded-s-none` |
| border-end-start-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:before:rounded-s-none` |

**`> *:is([data-slot=separator]:has(~button:hover):not(:has(~[data-slot=separator]~[data-slot]:hover)),[data-slot=separator]:has(~[data-slot][data-pressed]):not(:has(~[data-slot=separator]~[data-slot][data-pressed])))::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:[[data-slot=separator]:has(~button:hover):not(:has(~[data-slot=separator]~[data-slot]:hover)),[data-slot=separator]:has(~[data-slot][data-pressed]):not(:has(~[data-slot=separator]~[data-slot][data-pressed]))]:before:bg-input/64` |
| background-color | — / `#ffffff0d` | `input/64%` | `dark:*:[[data-slot=separator]:has(~button:hover):not(:has(~[data-slot=separator]~[data-slot]:hover)),[data-slot=separator]:has(~[data-slot][data-pressed]):not(:has(~[data-slot=separator]~[data-slot][data-pressed]))]:before:bg-input/64` |

**`> *:is(button:hover~[data-slot=separator]:not([data-slot]:hover~[data-slot=separator]~[data-slot=separator]),[data-slot][data-pressed]~[data-slot=separator]:not([data-slot][data-pressed]~[data-slot=separator]~[data-slot=separator]))::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:[button:hover~[data-slot=separator]:not([data-slot]:hover~[data-slot=separator]~[data-slot=separator]),[data-slot][data-pressed]~[data-slot=separator]:not([data-slot][data-pressed]~[data-slot=separator]~[data-slot=separator])]:before:bg-input/64` |
| background-color | — / `#ffffff0d` | `input/64%` | `dark:*:[button:hover~[data-slot=separator]:not([data-slot]:hover~[data-slot=separator]~[data-slot=separator]),[data-slot][data-pressed]~[data-slot=separator]:not([data-slot][data-pressed]~[data-slot=separator]~[data-slot=separator])]:before:bg-input/64` |

**`> *[data-slot]:has(~[data-slot])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:rounded-e-none` |
| border-end-end-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:rounded-e-none` |
| border-inline-end-style | `solid` | `var(--tw-border-style)` | `*:data-slot:has-[~[data-slot]]:border-e-0` |
| border-inline-end-width | `0px` | `0px` | `*:data-slot:has-[~[data-slot]]:border-e-0` |

**`> *[data-slot]:has(~[data-slot])::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:data-slot:has-[~[data-slot]]:before:rounded-e-none` |
| border-start-end-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:before:rounded-e-none` |
| border-end-end-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:before:rounded-e-none` |

**`> *[data-slot]:not(*[data-slot="separator"]):has(~[data-slot])::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-end-[0.5px]` |
| inset-inline-end | `-0.5px` | `calc(0.5px * -1)` | `*:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-end-[0.5px]` |

#### groupVariants: orientation=vertical

```txt
flex-col *:pointer-coarse:after:min-h-auto *:data-slot:has-[~[data-slot]]:rounded-b-none *:data-slot:has-[~[data-slot]]:border-b-0 *:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-bottom-[0.5px] *:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:hidden *:data-slot:has-[~[data-slot]]:before:rounded-b-none dark:*:last:before:hidden dark:*:first:before:block *:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-top-[0.5px] *:[[data-slot]~[data-slot]]:rounded-t-none *:[[data-slot]~[data-slot]]:border-t-0 *:[[data-slot]~[data-slot]]:before:rounded-t-none
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-direction | `column` | `column` | `flex-col` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(>[data-slot=group]) | gap | `8px` | `has-[>[data-slot=group]]:gap-2` |

**`> *::after` — mobile differences**: content `""` · min-height `auto`

**`> *:first-child::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:first:before:block` |
| display | — / `block` | `block` | `dark:*:first:before:block` |

**`> *:is([data-slot]~[data-slot]:not([data-slot=separator]))::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-start | `(unset)` | `` | `(removed)` |
| top | `-0.5px` | `calc(0.5px * -1)` | `*:[[data-slot]~[data-slot]:not([data-slot=separator])]:before:-top-[0.5px]` |

**`> *:is([data-slot]~[data-slot])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `(unset)` | `` | `(removed)` |
| border-end-start-radius | `(unset)` | `` | `(removed)` |
| border-inline-start-style | `(unset)` | `` | `(removed)` |
| border-inline-start-width | `(unset)` | `` | `(removed)` |
| border-top-left-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:rounded-t-none` |
| border-top-right-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:rounded-t-none` |
| border-top-style | `solid` | `var(--tw-border-style)` | `*:[[data-slot]~[data-slot]]:border-t-0` |
| border-top-width | `0px` | `0px` | `*:[[data-slot]~[data-slot]]:border-t-0` |

**`> *:is([data-slot]~[data-slot])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `(unset)` | `` | `(removed)` |
| border-end-start-radius | `(unset)` | `` | `(removed)` |
| border-top-left-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:before:rounded-t-none` |
| border-top-right-radius | `0` | `0` | `*:[[data-slot]~[data-slot]]:before:rounded-t-none` |

**`> *:last-child::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:last:before:hidden` |
| display | — / `none` | `none` | `dark:*:last:before:hidden` |

**`> *[data-slot]:has(~[data-slot])` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `(unset)` | `` | `(removed)` |
| border-end-end-radius | `(unset)` | `` | `(removed)` |
| border-inline-end-style | `(unset)` | `` | `(removed)` |
| border-inline-end-width | `(unset)` | `` | `(removed)` |
| border-bottom-right-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:rounded-b-none` |
| border-bottom-left-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:rounded-b-none` |
| border-bottom-style | `solid` | `var(--tw-border-style)` | `*:data-slot:has-[~[data-slot]]:border-b-0` |
| border-bottom-width | `0px` | `0px` | `*:data-slot:has-[~[data-slot]]:border-b-0` |

**`> *[data-slot]:has(~[data-slot])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `(unset)` | `` | `(removed)` |
| border-end-end-radius | `(unset)` | `` | `(removed)` |
| border-bottom-right-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:before:rounded-b-none` |
| border-bottom-left-radius | `0` | `0` | `*:data-slot:has-[~[data-slot]]:before:rounded-b-none` |

**`> *[data-slot]:not(*[data-slot="separator"]):has(~[data-slot])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| inset-inline-end | `(unset)` | `` | `(removed)` |
| bottom | `-0.5px` | `calc(0.5px * -1)` | `*:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:-bottom-[0.5px]` |
| display | `none` | `none` | `*:data-slot:not-data-[slot=separator]:has-[~[data-slot]]:before:hidden` |

### `GroupText` → `[data-slot=group-text]` · `useRender <div>` (line 55)

#### base

```txt
relative inline-flex items-center gap-2 whitespace-nowrap rounded-lg border border-input bg-muted not-dark:bg-clip-padding px-[calc(--spacing(3)-1px)] text-base text-muted-foreground shadow-xs/5 outline-none transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/6%)] sm:text-sm dark:bg-input/64 dark:before:shadow-[0_-1px_--theme(--color-white/6%)] [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:-mx-0.5 [&_svg]:shrink-0
```

**element — default**

Key metrics: display `inline-flex` · padding-inline `11px` · gap `8px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#0000000a` / `#ffffff0d` (muted) · color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#0000000a` / `#ffffff0d` | `muted` | `bg-muted` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — mobile differences**: font-size `16px` · line-height `24px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/6%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000f` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/6%)]` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `-2px` | `calc(var(--spacing) * -0.5)` | `[&_svg]:-mx-0.5` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

**`svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:[&_svg:not([class*='size-'])]:size-4` |

**`svg:not([class*='size-'])` — mobile differences**: width `18px` · height `18px`

### `GroupSeparator` → (no data-slot) · `<Separator>` (line 77)

Attributes: `orientation={orientation}`

Prop defaults: `orientation="vertical"`

#### base

```txt
pointer-events-none relative z-2 bg-input before:absolute before:inset-0 has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:translate-x-px has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:bg-ring dark:before:bg-input/32 [[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+&,[data-slot=number-field]:focus-within+input+&]:bg-ring [[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+input+&]:-translate-x-px
```

**element — default**

Key metrics: background-color `#0000001a` / `#ffffff14` (input) · z-index `2`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `relative` | `relative` | `relative` |
| z-index | `2` | `2` | `z-2` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(+[data-slot=input-control]:focus-within) | background-color | `#a1a1a1` / `#737373` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:bg-ring` |
| has(+[data-slot=input-control]:focus-within) | translate | `1px 0` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:translate-x-px` |
| has(+[data-slot=input-group]:focus-within) | background-color | `#a1a1a1` / `#737373` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:bg-ring` |
| has(+[data-slot=input-group]:focus-within) | translate | `1px 0` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:translate-x-px` |
| has(peer(+[data-slot=select-trigger]:focus-visible)) | background-color | `#a1a1a1` / `#737373` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:bg-ring` |
| has(peer(+[data-slot=select-trigger]:focus-visible)) | translate | `1px 0` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:translate-x-px` |
| has(+[data-slot=number-field]:focus-within) | background-color | `#a1a1a1` / `#737373` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:bg-ring` |
| has(+[data-slot=number-field]:focus-within) | translate | `1px 0` | `has-[+[data-slot=input-control]:focus-within,+[data-slot=input-group]:focus-within,+[data-slot=select-trigger]:focus-visible+*,+[data-slot=number-field]:focus-within]:translate-x-px` |
| prev([data-slot=input-control]:focus-within) | background-color | `#a1a1a1` / `#737373` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+&,[data-slot=number-field]:focus-within+input+&]:bg-ring` |
| prev([data-slot=input-control]:focus-within) | translate | `-1px 0` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+input+&]:-translate-x-px` |
| prev([data-slot=input-group]:focus-within) | background-color | `#a1a1a1` / `#737373` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+&,[data-slot=number-field]:focus-within+input+&]:bg-ring` |
| prev([data-slot=input-group]:focus-within) | translate | `-1px 0` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+input+&]:-translate-x-px` |
| prev([data-slot=select-trigger]:focus-visible+*) | background-color | `#a1a1a1` / `#737373` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+&,[data-slot=number-field]:focus-within+input+&]:bg-ring` |
| prev([data-slot=select-trigger]:focus-visible+*) | translate | `-1px 0` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+input+&]:-translate-x-px` |
| prev([data-slot=number-field]:focus-within) | background-color | `#a1a1a1` / `#737373` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+&,[data-slot=number-field]:focus-within+input+&]:bg-ring` |
| prev([data-slot=number-field]:focus-within+input) | background-color | `#a1a1a1` / `#737373` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+&,[data-slot=number-field]:focus-within+input+&]:bg-ring` |
| prev([data-slot=number-field]:focus-within+input) | translate | `-1px 0` | `[[data-slot=input-control]:focus-within+&,[data-slot=input-group]:focus-within+&,[data-slot=select-trigger]:focus-visible+*+&,[data-slot=number-field]:focus-within+input+&]:-translate-x-px` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:inset-0` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| background-color | — / `#ffffff07` | `input/32%` | `dark:before:bg-input/32` |

